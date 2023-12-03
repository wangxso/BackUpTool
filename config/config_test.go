package config_test

import (
	"log"
	"testing"

	"github.com/wangxso/backuptool/config"
)

func TestReader(t *testing.T) {
	conf := config.Reader()
	log.Fatalf("Config is %v", conf)
}
