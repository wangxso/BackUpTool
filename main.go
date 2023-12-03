package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/download"
)

func main() {
	//  auth.Login()
	// targetPath := ""
	// sourcePath := ""
	// upload.Upload(targetPath, sourcePath)
	redis_cli := db.Client
	accessToken, _ := redis_cli.Get(redis_cli.Context(), "AccessCode").Result()
	resp := download.GetFileList(accessToken)
	fids := make([]uint64, 0)
	list := resp.List
	for _, v := range list {
		fids = append(fids, uint64(v.FsId))
	}
	dlinks, err := download.GetDlink(accessToken, fids)
	if err != nil {
		logrus.Error(err)
	}
	targetPath := "/Users/wangxs/Downloads"
	for _, v := range dlinks {
		uri := fmt.Sprintf("%s&access_token=%s", v["dlink"], accessToken)
		filename := v["filename"]
		download.Download(uri, filename, targetPath)
	}

}
