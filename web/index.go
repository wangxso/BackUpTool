package web

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wangxso/backuptool/auth"
	"github.com/wangxso/backuptool/cloudsync"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
)

func StartWeb() {
	r := gin.Default()
	r.GET("/sync/status", UploadStatus)
	r.GET("/sync", SyncFolder)
	r.GET("/auth", Auth)
	r.GET("/login", AuthLogin)
	r.GET("/cache", CacheFileMD5Handler)
	r.GET("/alive", AliveHandler)
	r.Run("0.0.0.0:8080")
}

func AliveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "alive",
	})

}

func Auth(c *gin.Context) {
	appKey := config.BackUpConfig.BaiduDisk.AppKey
	deviceId := config.BackUpConfig.BaiduDisk.SecretKey
	url := fmt.Sprintf("http://openapi.baidu.com/oauth/2.0/authorize?response_type=code&client_id=%s&redirect_uri=oob&scope=basic,netdisk&device_id=%s", appKey, deviceId)
	c.JSON(http.StatusOK, gin.H{
		"message": "Please visit",
		"url":     url,
	})
}

func AuthLogin(c *gin.Context) {
	code := c.Query("code")
	resp := auth.Login(code)
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"token":   resp.AccessToken,
		"refresh": resp.RefreshToken,
		"scope":   resp.Scope,
		"expires": resp.ExpiresIn,
	})
}

func CacheFileMD5Handler(c *gin.Context) {
	cloudsync.CacheFileMD5Map()
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}

func SyncFolder(c *gin.Context) {
	err := cloudsync.SyncFolder()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}

func UploadStatus(c *gin.Context) {
	redisCli := db.Client
	uploadMap, err := redisCli.HGetAll(redisCli.Context(), cloudsync.UPLOAD_PATHS).Result()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	uploadedFileList := make([]string, 0)
	unuploadFileList := make([]string, 0)
	for _, k := range uploadMap {
		if uploadMap[k] == "true" {
			uploadedFileList = append(uploadedFileList, k)
		} else {
			unuploadFileList = append(unuploadFileList, k)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"uploadedFileList": uploadedFileList,
		"unuploadFileList": unuploadFileList,
	})
}
