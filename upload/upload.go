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
	"strconv"

	"github.com/sirupsen/logrus"
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

func UploadSlice(accessToken string, partseq string, path string, uploadid string, type_ string, file *os.File) []byte {
	configuration := openapiclient.NewConfiguration()
	//configuration.Debug = true
	api_client := openapiclient.NewAPIClient(configuration)
	resp, r, err := api_client.FileuploadApi.Pcssuperfile2(context.Background()).AccessToken(accessToken).Partseq(partseq).Path(path).Uploadid(uploadid).Type_(type_).File(file).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileuploadApi.Pcssuperfile2``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// response from `Pcssuperfile2`: string
	logrus.Info("Response from `FileuploadApi.Pcssuperfile2`: ", resp)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}

	return bodyBytes
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
		chunkFileName := fmt.Sprintf("%s.%d", filepath, chunkCount)
		chunkFile, err := os.Create(chunkFileName)
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

func deleteChunks(filePath string) error {
	// 分片计数器
	chunkCount := 0

	for {
		// 构造分片文件名
		chunkFileName := fmt.Sprintf("%s.%d", filePath, chunkCount)
		// logrus.Info("delete file", chunkFileName)
		// 尝试删除分片文件
		err := os.Remove(chunkFileName)
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

func Upload(targetPath, sourcePath string) {
	redis_cli := db.Client
	path := targetPath
	source := sourcePath
	accessCode, _ := redis_cli.Get(redis_cli.Context(), "AccessCode").Result()
	isdir := int32(0)
	autoinit := int32(1)
	blockList, size, err := spiltFile(source)
	if err != nil {
		logrus.Error(err)
	}
	blockListByte, err := json.Marshal(blockList)
	blockListStr := string(blockListByte)
	if err != nil {
		logrus.Error(err)
	}
	preCreateResp := PreCreateUpload(accessCode, path, isdir, int32(size), autoinit, string(blockListStr), 3)
	for i := 0; i < len(blockList); i++ {
		slicePath := fmt.Sprintf("%s.%d", source, i)
		file, err := os.Open(slicePath)
		if err != nil {
			logrus.Error(err)
		}
		UploadSlice(accessCode, strconv.Itoa(i), path, preCreateResp.Uploadid, "tmpfile", file)
	}
	UploadCreate(accessCode, path, isdir, int32(size), preCreateResp.Uploadid, blockListStr, 3)
	defer deleteChunks(source)
}
