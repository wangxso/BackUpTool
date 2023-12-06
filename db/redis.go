package db

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
)

var (
	Client *redis.Client
)

func LoadRedis() {
	Addr := fmt.Sprintf("%s:%s", config.BackUpConfig.Redis.Host, config.BackUpConfig.Redis.Port)
	Client = redis.NewClient(&redis.Options{
		Addr:     Addr,                               // Redis 服务器地址
		Password: config.BackUpConfig.Redis.Password, // Redis 服务器密码（如果有的话）
		DB:       config.BackUpConfig.Redis.Db,       // 使用的 Redis 数据库索引
	})
	_, err := Client.Ping(Client.Context()).Result()

	if err != nil {
		if err == redis.Nil {
			logrus.Error("Redis 服务器未启动")
		} else {
			logrus.Error("无法连接到 Redis:", err)
		}
		return
	}
	logrus.Info("Connect to Redis successfully")
}
