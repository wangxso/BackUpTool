package clousync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/download"
	"github.com/wangxso/backuptool/upload"
	"github.com/wangxso/backuptool/utils"
)

func SyncFolder() error {
	waitingCount := 0
	uploadCount := 0
	downloadCount := 0
	skipCount := 0
	sourceFolder := config.BackUpConfig.General.SyncDir
	targetFolder := config.BackUpConfig.BaiduDisk.SyncDir
	fidMap := make(map[string]uint64)
	redisCli := db.Client
	accessToken, _ := redisCli.Get(redisCli.Context(), "AccessCode").Result()
	// 获取云端文件
	cloudFileList := make([]download.FileItem, 0)
	resp := download.GetMultiFileList(accessToken, targetFolder, 1, "time", 0, 0, 1000)
	if resp.Errno != -9 {
		cloudFileList = append(cloudFileList, resp.List...)
	}
	// 31066错误为文件不存在
	if resp.Errno != 0 && resp.Errno != 31066 {
		logrus.Error("ErrorNo: ", resp.Errmsg)
		logrus.Error("ErrorMsg: ", resp.RequestID)
		return errors.New("ErrorNo: " + fmt.Sprint(resp.Errno) + " Errormsg: " + fmt.Sprint(resp.Errmsg))
	}
	// 一次获取1000个目录，如果有剩余，继续获取
	for resp.HasMore == 1 {
		resp = download.GetMultiFileList(accessToken, targetFolder, 1, "time", 0, resp.Cursor, 1000)
		if resp.Errno != -9 {
			cloudFileList = append(cloudFileList, resp.List...)
		}
	}
	couldMd5FileMap := make(map[string]string)

	for _, v := range cloudFileList {
		if v.IsDir == 0 {
			couldMd5FileMap[v.ServerFilename] = v.MD5
			fidMap[v.ServerFilename] = uint64(v.FsID)
		}
	}

	// 找到sourceFolder下的所有文件
	// 傻逼百度，云端哈希不是真实文件的哈希
	// 递归遍历文件
	sourceFileMap := make(map[string]string)

	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Error(err)
			return err
		}

		if info.IsDir() {
			return nil // 继续遍历子目录
		}
		waitingCount++
		filename := info.Name()
		relativePath, err := utils.GetRelativeSubdirectory(sourceFolder, path)
		if err != nil {
			logrus.Error(err)
			return err
		}
		sourceMD5, _ := utils.CalculateMD5(path)
		sourceFileMap[filename] = "true"
		// 对比目录差异
		// 不在云端的上传
		if relativePath == "." {
			relativePath = ""
		}

		// if _, ok := couldMd5FileMap[filename]; !ok {
		// 上传文件
		cloudMD5 := couldMd5FileMap[filename]
		targetMD5, _ := redisCli.Get(redisCli.Context(), cloudMD5).Result()
		if sourceMD5 != targetMD5 {
			logrus.Info("filename: ", filename, " md5: ", sourceMD5, " Upload File")
			uploadCount++
			upload.Upload(relativePath, path)

		} else {
			logrus.Info("filename: ", filename, " md5: ", sourceMD5, " File Exsist, Skip Upload")
			skipCount++
		}
		return nil
	})

	if err != nil {
		logrus.Error("Error reading directory: ", err)
		return errors.New("Error reading directory: " + err.Error())
	}

	// 下载本地没有的文件
	for path := range couldMd5FileMap {
		if _, ok := sourceFileMap[path]; !ok {
			downloadCount++
			logrus.Infof("Download Source File Name [%s]", path)
			download.Download(fidMap[path], sourceFolder)
		}
	}
	// 上传这些文件

	// 下载没有的文件
	logrus.Info("Waiting Count: ", waitingCount, " Upload Count: ", uploadCount, " Download Count: ", downloadCount, " Skip Count: ", skipCount, " CloudFile Count: ", len(couldMd5FileMap))
	return nil
}
