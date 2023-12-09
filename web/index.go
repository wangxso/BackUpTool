package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wangxso/backuptool/auth"
	"github.com/wangxso/backuptool/cloudsync"
	"github.com/wangxso/backuptool/db"
)

func StartWeb() {
	r := gin.Default()
	r.GET("/sync/status", UploadStatus)
	r.GET("/sync", SyncFolder)
	r.GET("/auth", AuthLogin)
	r.GET("/cache", CacheFileMD5Handler)
	r.GET("/alive", AliveHandler)
	r.Run("0.0.0.0:8080")
}

func AliveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "alive",
	})

}

func AuthLogin(c *gin.Context) {
	auth.Login()

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
