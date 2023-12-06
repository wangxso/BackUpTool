package main

import (
	"flag"

	"github.com/wangxso/backuptool/auth"
	clousync "github.com/wangxso/backuptool/cloudsync"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/handler"
)

const (
	DEFAULT_CONFIG_PATH = "./config.yaml"
)

var (
	configPath string
	authFlag   bool
	syncFlag   bool
)

func init() {
	flag.StringVar(&configPath, "config", DEFAULT_CONFIG_PATH, "config file path")
	flag.BoolVar(&authFlag, "auth", false, "Is Open Auth Mode(default false)")
	flag.BoolVar(&syncFlag, "sync", false, "Is Sync Mode(default false)")
}

func main() {
	defer handler.HandlerGlobalErrors()

	flag.Parse()
	config.LoadConfig(DEFAULT_CONFIG_PATH)
	db.LoadRedis()
	// 登录模式
	if authFlag {
		auth.Login()
		return
	}
	// 同步模式
	if syncFlag {
		clousync.SyncFolder()
		return
	}
}
