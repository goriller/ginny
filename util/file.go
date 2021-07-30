package util

import (
	"os"
	"path/filepath"
)

// GetCurrentDir 获取当前所在目录
func GetCurrentDir() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return dir, nil
}

//FileExist 判断文件是否存在
func FileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateDirIfNotExist 创建目录如果不存在
func CreateDirIfNotExist(path string) bool {
	success := true
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				success = false
			}
		} else {
			success = false
		}
	}
	return success
}

//CreateFileIfNotExist 创建文件如果不存在
func CreateFileIfNotExist(filePath string) error {
	_, err := os.Stat(filePath) //os.Stat获取文件信息
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		_ = CreateDirIfNotExist(filepath.Dir(filePath))
		fd, err := os.Create(filePath)
		defer func() {
			if fd != nil {
				fd.Close()
			}
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
