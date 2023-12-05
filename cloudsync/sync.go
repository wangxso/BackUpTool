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

	sourceFolder := config.BackUpConfig.General.SyncDir
	targetFolder := config.BackUpConfig.BaiduDisk.SyncDir
	fidMap := make(map[string]uint64)
	redisCli := db.Client
	accessToken, _ := redisCli.Get(redisCli.Context(), "AccessCode").Result()
	resp := download.GetFileList(accessToken, targetFolder, "time", "0", "0", 10, 1)

	couldMd5FileMap := make(map[string]string)
	if resp.ErrorNo != -9 {
		cloudFileList := resp.List
		for _, v := range cloudFileList {
			couldMd5FileMap[v.ServerFileName] = v.MD5
			fidMap[v.ServerFileName] = uint64(v.FsId)
			logrus.Info("filename: ", v.ServerFileName, " md5: ", v.MD5)
		}
	}

	if resp.ErrorNo != 0 && resp.ErrorNo != -9 {
		logrus.Error("ErrorNo: ", resp.ErrorNo)
		logrus.Error("ErrorMsg: ", resp.RequestId)
		return errors.New("ErrorNo: " + fmt.Sprint(resp.ErrorNo) + " RequestId: " + fmt.Sprint(resp.RequestId))
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

		filename := info.Name()
		relativePath, err := utils.GetRelativeSubdirectory(sourceFolder, path)
		// sourceMD5, err := calculateMD5(filepath.Join(dir, filename))
		if err != nil {
			logrus.Error(err)
			return err
		}
		sourceFileMap[filename] = "true"
		// 对比目录差异
		// 不在云端的上传
		if relativePath == "." {
			relativePath = ""
		}
		if _, ok := couldMd5FileMap[filename]; !ok {
			// 下载文件
			logrus.Info("Start Sync File(Upload): ", relativePath+filename)

			uploadPath := filepath.Join(targetFolder, filepath.Join(relativePath, filename))
			upload.Upload(uploadPath, path)
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
			logrus.Infof("Source File Name [%s]", path)
			download.Download(fidMap[path], sourceFolder)
		}
	}
	// 上传这些文件

	// 下载没有的文件
	return nil
}
