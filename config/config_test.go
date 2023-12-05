package config_test

import (
	"log"
	"testing"

	"github.com/wangxso/backuptool/config"
)

func TestReader(t *testing.T) {
	log.Fatalf("Config is %v", config.BackUpConfig)
}
