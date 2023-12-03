package upload_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/db"
	"github.com/wangxso/backuptool/upload"
)

func TestPreCreateUpload(t *testing.T) {
	redis_cli := db.Client
	acessToken, _ := redis_cli.Get(redis_cli.Context(), "AccessCode").Result()
	resp := upload.PreCreateUpload(acessToken, "1.txt", 0, 64, 1, "['98d02a0f542220']", 3)
	log.Info(resp)
}
