package main

import (
	clousync "github.com/wangxso/backuptool/cloudsync"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
)

const (
	DEFAULT_CONFIG_PATH = "./config.yaml"
)

func main() {
	config.LoadConfig(DEFAULT_CONFIG_PATH)
	db.LoadRedis()
	clousync.SyncFolder()
}
