package upload

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	openapiclient "github.com/wangxso/backuptool/openxpanapi"
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

func PreCreateUpload(accessToken string, path string, isdir int32, size int32, autoinit int32, blockList string, rtype int32) precreateReturnType {
	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	resp, r, err := api_client.FileuploadApi.Xpanfileprecreate(context.Background()).AccessToken(accessToken).Path(path).Isdir(isdir).Size(size).Autoinit(autoinit).BlockList(blockList).Rtype(rtype).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Xpanfileprecreate``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Xpanfileprecreate`: Fileprecreateresponse
	logrus.Info("Response from `FileuploadApi.Xpanfileprecreate`: ", resp)

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
	resp, r, err := api_client.FileuploadApi.Pcssuperfile2(context.Background()).AccessToken(accessToken).Partseq(partseq).Path(path_).Uploadid(uploadid).Type_(type_).File(file).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Pcssuperfile2``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Pcssuperfile2`: string
	logrus.Info("Response from `FileuploadApi.Pcssuperfile2`: ", resp)

	_, err = io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}
	defer deleteOneChunks(path.Base(path_))
	return err
}

func UploadCreate(accessToken string, path string, isdir int32, size int32, uploadid string, blockList string, rtype int32) {
	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	resp, r, err := api_client.FileuploadApi.Xpanfilecreate(context.Background()).AccessToken(accessToken).Path(path).Isdir(isdir).Size(size).Uploadid(uploadid).BlockList(blockList).Rtype(rtype).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Xpanfilecreate``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Xpanfilecreate`: Filecreateresponse
	logrus.Info("Response from `FileuploadApi.Xpanfilecreate`: ", resp)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}

	fmt.Println(string(bodyBytes))
}

func spiltFile(filepath string) ([]string, int, error) {
	file, err := os.Open(filepath)
	blockList := make([]string, 0)
	if err != nil {
		return nil, 0, err
	}
	stat, _ := file.Stat()
	size := stat.Size()
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
		chunkFileName := fmt.Sprintf("%s.%d", path.Base(filepath), chunkCount)
		chunkFilePath := path.Join(config.BackUpConfig.General.TmpDir, chunkFileName)
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
	chunkFilePath := path.Join(config.BackUpConfig.General.TmpDir, fileName)
	// logrus.Info("delete file", chunkFileName)
	// 尝试删除分片文件
	err := os.Remove(chunkFilePath)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func deleteChunks(fileName string) error {
	// 分片计数器
	chunkCount := 0

	for {
		// 构造分片文件名
		chunkFileName := fmt.Sprintf("%s.%d", fileName, chunkCount)
		chunkFilePath := path.Join(config.BackUpConfig.General.TmpDir, chunkFileName)
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
func Upload(targetPath, sourcePath string) {
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
	// Create a channel to receive upload results
	uploadResultChan := make(chan error)

	// Upload each slice of the file
	for i := 0; i < len(blockList); i++ {
		go func(sliceIndex int) {
			slicePath := fmt.Sprintf("%s.%d", path.Base(sourcePath), sliceIndex)
			slicePath = path.Join(config.BackUpConfig.General.TmpDir, slicePath)
			file, err := os.Open(slicePath)
			if err != nil {
				uploadResultChan <- err
				return
			}
			err = UploadSlice(accessCode, strconv.Itoa(sliceIndex), targetPath, preCreateResp.Uploadid, "tmpfile", file)
			// Create the final upload
			uploadResultChan <- err
		}(i)
	}

	// Wait for all uploads to complete
	for i := 0; i < len(blockList); i++ {
		err := <-uploadResultChan
		if err != nil {
			slicePath := fmt.Sprintf("%s.%d", path.Base(sourcePath), i)
			deleteChunks(slicePath)
			panic(err.Error())
		}
	}

	UploadCreate(accessCode, targetPath, isDir, int32(size), preCreateResp.Uploadid, blockListStr, 3)

	// Clean up the chunks
	defer deleteChunks(path.Base(sourcePath))
}
