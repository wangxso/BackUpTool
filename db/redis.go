package db

import (
	"fmt"

	"github.com/go-redis/redis/v8"
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
}
