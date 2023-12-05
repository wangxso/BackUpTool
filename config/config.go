package config

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	BaiduDisk struct {
		AppKey      string `yaml:"AppKey"`
		SecretKey   string `yaml:"SecretKey"`
		SyncDir     string `yaml:"syncDir"`
		RedirectUri string `yaml:"RedirectUri"`
	} `yaml:"BaiduDisk"`

	General struct {
		Debug   bool   `yaml:"debug"`
		TmpDir  string `yaml:"tmpDir"`
		SyncDir string `yaml:"syncDir"`
	} `yaml:"General"`

	Redis struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Password string `yaml:"password"`
		Db       int    `yaml:"db"`
	} `yaml:"Redis"`
}

var BackUpConfig Config

func LoadConfig(configPath string) {

	yamlFile, err := os.ReadFile(configPath)

	if err != nil {
		logrus.Error("Failed to read YAML file: ", err)
	}

	err = yaml.Unmarshal(yamlFile, &BackUpConfig)
	if err != nil {
		logrus.Error("Failed to unmarshal YAML ", err)
	}
}
