package clousync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/download"
	"github.com/wangxso/backuptool/upload"
	"github.com/wangxso/backuptool/utils"
)

const (
	DOWNLOAD_PATHS = "download_paths"
	UPLOAD_PATHS   = "upload_paths"
)

// SyncFolder synchronizes the source folder with the target folder in the BaiduDisk cloud storage.
//
// It retrieves the cloud file list, compares it with the local files in the source folder, and performs the following operations:
// - Uploads the files that are not present in the cloud storage.
// - Downloads the files that are present in the cloud storage but missing locally.
//
// The function takes no parameters and returns an error if any occurs during the synchronization process.
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

	// 初始化所需的变量和锁
	var wg sync.WaitGroup
	// var mu sync.Mutex

	// 计算所有需要上传的文件path
	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Error(err)
			return err
		}

		if info.IsDir() {
			return nil // 继续遍历子目录
		}
		waitingCount++
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			filename := info.Name()
			relativePath, err := utils.GetRelativeSubdirectory(sourceFolder, path)
			if err != nil {
				logrus.Error(err)
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
				// upload.Upload(relativePath, path)
				redisCli.HSet(redisCli.Context(), DOWNLOAD_PATHS, relativePath, false)
			} else {
				logrus.Info("filename: ", filename, " md5: ", sourceMD5, " File Exsist, Skip Upload")
			}
		}(path)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logrus.Error("Error reading directory: ", err)
		return errors.New("Error reading directory: " + err.Error())
	}
	wg.Wait()
	// 下载本地没有的文件
	for path := range couldMd5FileMap {
		if _, ok := sourceFileMap[path]; !ok {
			logrus.Infof("Download Source File Name [%s]", path)
			redisCli.HSet(redisCli.Context(), DOWNLOAD_PATHS, fidMap[path], false)
		}
	}
	// 上传或者下载这些文件
	// 上传
	for path := range redisCli.HGetAll(redisCli.Context(), UPLOAD_PATHS).Val() {
		if redisCli.HGet(redisCli.Context(), UPLOAD_PATHS, path).Val() == "false" {
			uploadCount++
			logrus.Infof("Upload Source File Name [%s]", path)
			if err != nil {
				return err
			}
			upload.Upload(path, sourceFolder)
			uploadCount++
			redisCli.HSet(redisCli.Context(), UPLOAD_PATHS, path, true)
		}
	}
	// 下载
	for path := range redisCli.HGetAll(redisCli.Context(), DOWNLOAD_PATHS).Val() {
		if redisCli.HGet(redisCli.Context(), DOWNLOAD_PATHS, path).Val() == "false" {
			uploadCount++
			logrus.Infof("Upload Source File Name [%s]", path)
			fid, err := strconv.ParseUint(path, 10, 64)
			if err != nil {
				return err
			}
			download.Download(fid, sourceFolder)
			downloadCount++
			redisCli.HSet(redisCli.Context(), DOWNLOAD_PATHS, path, true)
		}
	}

	logrus.Info("Waiting Count: ", waitingCount, " Upload Count: ", uploadCount, " Download Count: ", downloadCount, " Skip Count: ", skipCount, " CloudFile Count: ", len(couldMd5FileMap))
	return nil
}
