package upload_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/karrick/godirwalk"
	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/utils"
)

func BenchmarkCountFiles(b *testing.B) {
	for n := 0; n < b.N; n++ {
		config.LoadConfig("/Users/wangxs/backuptool/config.yaml")
		logrus.Info(countFiles(config.BackUpConfig.General.SyncDir))
	}
}

func BenchmarkGodirWalk(b *testing.B) {
	for n := 0; n < b.N; n++ {
		config.LoadConfig("/Users/wangxs/backuptool/config.yaml")
		walkDir(config.BackUpConfig.General.SyncDir)
	}
}

func countFiles(dir string) (int, error) {
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件而不是文件夹，则增加计数
		if !info.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return count, nil
}

func walkDir(path string) {
	dir := path // 要遍历的目录路径

	err := godirwalk.Walk(dir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			if !de.IsDir() {
				md5, err := utils.CalculateMD5(path)
				if err != nil {
					fmt.Print(err)
				}

				fmt.Println(path, " md5: ", md5) // 输出文件路径
			}
			return nil
		},
		ErrorCallback: func(path string, err error) godirwalk.ErrorAction {
			fmt.Printf("Error walking %s: %v\n", path, err)
			return godirwalk.SkipNode
		},
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
}
