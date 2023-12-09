package main

import (
	_ "net/http/pprof"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/handler"
	"github.com/wangxso/backuptool/web"
)

const (
	DEFAULT_CONFIG_PATH = "./config.yaml"
)

func main() {
	defer handler.HandlerGlobalErrors()
	defer db.CloseRedis()
	// 创建日志文件
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatal(err)
	}
	defer logFile.Close()

	// 设置 logrus 输出为文件
	logrus.SetOutput(logFile)

	// 设置 logrus 格式
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
	})

	config.LoadConfig(DEFAULT_CONFIG_PATH)
	db.LoadRedis()
	web.StartWeb()
}
