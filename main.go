package main

import (
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
	// 创建一个新的日志记录器实例
	logger := logrus.New()
	// 创建日志文件
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	// 设置文件日志钩子为日志记录器的输出
	logger.SetOutput(logFile)

	// 设置控制台日志钩子为日志记录器的输出

	config.LoadConfig(DEFAULT_CONFIG_PATH)
	db.LoadRedis()
	web.StartWeb()
}
