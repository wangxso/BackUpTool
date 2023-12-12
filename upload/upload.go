package upload

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	openapiclient "github.com/wangxso/backuptool/openxpanapi"
	"github.com/wangxso/backuptool/utils"
)

const (
	chunkSize = 1024 * 1024 * 4 // 4MB
)

type precreateReturnType struct {
	Path       string        `json:"path"`
	Uploadid   string        `json:"uploadid"`
	ReturnType int           `json:"return_type"`
	BlockList  []interface{} `json:"block_list"`
	Errno      int           `json:"errno"`
	RequestID  int64         `json:"request_id"`
}

type createFileReturnType struct {
	Errno          int    `json:"errno"`
	FsID           int64  `json:"fs_id"`
	MD5            string `json:"md5"`
	ServerFilename string `json:"server_filename"`
	Category       int    `json:"category"`
	Path           string `json:"path"`
	Size           uint64 `json:"size"`
	Ctime          int64  `json:"ctime"`
	Mtime          int64  `json:"mtime"`
	IsDir          int    `json:"isdir"`
	Name           string `json:"name"`
}

type UploadSmallFileReturn struct {
	Ctime     int64  `json:"ctime"`
	FsID      int64  `json:"fs_id"`
	MD5       string `json:"md5"`
	Mtime     int64  `json:"mtime"`
	Path      string `json:"path"`
	RequestID int64  `json:"request_id"`
	Size      int64  `json:"size"`
}

func PreCreateUpload(accessToken string, path string, isdir int32, size int32, autoinit int32, blockList string, rtype int32) precreateReturnType {
	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	_, r, err := api_client.FileuploadApi.Xpanfileprecreate(context.Background()).AccessToken(accessToken).Path(path).Isdir(isdir).Size(size).Autoinit(autoinit).BlockList(blockList).Rtype(rtype).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Xpanfileprecreate``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Xpanfileprecreate`: Fileprecreateresponse
	// logrus.Info("Response from `FileuploadApi.Xpanfileprecreate`: ", resp)

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}
	var response precreateReturnType
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func UploadSlice(accessToken string, partseq string, path_ string, uploadid string, type_ string, file *os.File) error {
	configuration := openapiclient.NewConfiguration()
	//configuration.Debug = true
	api_client := openapiclient.NewAPIClient(configuration)
	_, r, err := api_client.FileuploadApi.Pcssuperfile2(context.Background()).AccessToken(accessToken).Partseq(partseq).Path(path_).Uploadid(uploadid).Type_(type_).File(file).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Pcssuperfile2``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Pcssuperfile2`: string
	// logrus.Info("Response from `FileuploadApi.Pcssuperfile2`: ", resp)

	_, err = io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}
	defer deleteOneChunks(filepath.Base(path_))
	return err
}

func UploadCreate(accessToken string, path string, isdir int32, size int32, uploadid string, blockList string, rtype int32) createFileReturnType {
	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	_, r, err := api_client.FileuploadApi.Xpanfilecreate(context.Background()).AccessToken(accessToken).Path(path).Isdir(isdir).Size(size).Uploadid(uploadid).BlockList(blockList).Rtype(rtype).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Xpanfilecreate``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Xpanfilecreate`: Filecreateresponse
	// logrus.Info("Response from `FileuploadApi.Xpanfilecreate`: ", resp)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}

	var response createFileReturnType
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func spiltFile(filePath string) ([]string, int, error) {
	file, err := os.Open(filePath)
	blockList := make([]string, 0)
	if err != nil {
		return nil, 0, err
	}
	stat, _ := file.Stat()
	size := stat.Size()
	filename := stat.Name()
	buffer := make([]byte, chunkSize)
	chunkCount := 0
	for {
		readBytes, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, int(size), err
		}
		chunkFileName := fmt.Sprintf("%s.%d", filename, chunkCount)
		chunkFilePath := filepath.Join(config.BackUpConfig.General.TmpDir, chunkFileName)
		chunkFile, err := os.Create(chunkFilePath)
		if err != nil {
			return nil, int(size), err
		}
		md5str := md5.Sum(buffer[:readBytes])
		blockList = append(blockList, hex.EncodeToString(md5str[:]))
		// logrus.Infof(hex.EncodeToString(md5str[:]))
		_, err = chunkFile.Write(buffer[:readBytes])
		if err != nil {
			return nil, int(size), err
		}
		chunkFile.Close()
		chunkCount++
	}
	return blockList, int(size), nil
}

func deleteOneChunks(fileName string) error {

	// 构造分片文件名
	chunkFilePath := filepath.Join(config.BackUpConfig.General.TmpDir, fileName)
	// logrus.Info("delete file", chunkFileName)
	// 尝试删除分片文件
	err := os.Remove(chunkFilePath)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func UploadSmallFile(accessToken, path, filePath string) (UploadSmallFileReturn, error) {
	var ret UploadSmallFileReturn
	host := "https://d.pcs.baidu.com"
	uri := fmt.Sprintf("%s/rest/2.0/pcs/file?method=upload&", host)
	// 读取文件上传

	params := url.Values{}
	params.Set("ondup", "overwrite")
	params.Set("path", path)
	params.Set("access_token", accessToken)
	uri += params.Encode()
	file, err := os.Open(filePath)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()
	// stat, _ := file.Stat()
	// filename := stat.Name()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "file")
	contentType := writer.FormDataContentType()
	headers := map[string]string{
		"Host":         host,
		"Content-Type": contentType,
	}
	if err != nil {
		panic(err.Error())

	}
	_, err = io.Copy(part, file)
	if err != nil {
		panic(err.Error())

	}
	err = writer.Close()
	if err != nil {
		panic(err.Error())

	}
	respBody, _, err := utils.SendHTTPRequest(uri, body, headers)
	if err != nil {
		return ret, err
	}
	if err = json.Unmarshal([]byte(respBody), &ret); err != nil {
		logrus.Error("[msg: unmarshal filemetas body failed] err:", err.Error())
		return ret, errors.New("unmarshal filemetas body failed,body")
	}
	logrus.Info(ret)
	return ret, nil
}

func deleteChunks(fileName string) error {
	// 分片计数器
	chunkCount := 0

	for {
		// 构造分片文件名
		chunkFileName := fmt.Sprintf("%s.%d", fileName, chunkCount)
		chunkFilePath := filepath.Join(config.BackUpConfig.General.TmpDir, chunkFileName)
		// logrus.Info("delete file", chunkFileName)
		// 尝试删除分片文件
		err := os.Remove(chunkFilePath)
		if err != nil {
			// 如果分片文件不存在，则表示已删除完所有分片
			if os.IsNotExist(err) {
				break
			}
			return err
		}

		chunkCount++
	}

	return nil
}

// Upload uploads a file from the sourcePath to the targetPath.
//
// Parameters:
// - targetPath: the path where the file will be uploaded.
// - sourcePath: the path of the file to be uploaded.
// Return type(s): None.
func Upload(targetPath, sourcePath string) (string, error) {
	// Get the Redis client from the db package
	redisCli := db.Client

	// Get the access code from Redis
	accessCode, _ := redisCli.Get(redisCli.Context(), "AccessCode").Result()

	// Initialize variables
	isDir := int32(0)
	autoInit := int32(1)

	// Split the file into blocks
	blockList, size, err := spiltFile(sourcePath)
	if err != nil {
		logrus.Error("[UploadSpiltFile]", err)
		panic(err.Error())
	}

	// Convert the blockList to JSON and store it as a string
	blockListByte, err := json.Marshal(blockList)
	blockListStr := string(blockListByte)
	if err != nil {
		logrus.Error("[BlockListMarshal] ", err)
		panic(err.Error())
	}

	// Pre-create the upload
	preCreateResp := PreCreateUpload(accessCode, targetPath, isDir, int32(size), autoInit, string(blockListStr), 3)
	var wg sync.WaitGroup
	errChan := make(chan error, len(blockList))
	for i := 0; i < len(blockList); i++ {
		slicePath := fmt.Sprintf("%s.%d", filepath.Base(sourcePath), i)
		slicePath = filepath.Join(config.BackUpConfig.General.TmpDir, slicePath)

		wg.Add(1)
		go UploadSliceAsync(&wg, accessCode, targetPath, preCreateResp.Uploadid, slicePath, i, len(blockList), errChan)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	var uploadErr error
	for err := range errChan {
		if err != nil {
			uploadErr = err
			break
		}
	}

	resp := UploadCreate(accessCode, targetPath, isDir, int32(size), preCreateResp.Uploadid, blockListStr, 3)
	if resp.Errno == 0 {
		// 上传成功
		return resp.MD5, uploadErr
	}

	// Clean up the chunks
	defer deleteChunks(filepath.Base(sourcePath))
	return "", nil
}

func UploadSliceAsync(wg *sync.WaitGroup, accessCode, targetPath, uploadID, slicePath string, index int, length int, errChan chan<- error) {
	defer wg.Done()

	file, err := os.Open(slicePath)
	if err != nil {
		errChan <- err
		return
	}
	defer file.Close()

	err = UploadSlice(accessCode, strconv.Itoa(index), targetPath, uploadID, "tmpfile", file)
	if err != nil {
		errChan <- err
		return
	}

	logrus.Infof("[UploadSlice] %d/%d\n", index, length)
}
