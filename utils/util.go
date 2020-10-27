package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	"git.code.oa.com/Ginny/ginny/loggy"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

// GenerateID 雪花ID uint64 -> string
func GenerateID() (string, error) {
	// 雪花
	node, err := snowflake.NewNode(1)
	if err != nil {
		loggy.Error("GetSnowID NewNode error",
			zap.Error(err),
		)
		return "", err
	}

	// Generate a snowflake ID.
	idSnow := node.Generate()
	id := idSnow.String()
	return id, nil
}

// IsEmptyString 为空判断
func IsEmptyString(text string) bool {
	s := strings.TrimSpace(text)
	return len(s) <= 0
}

// IsSpaceOrEmpty 判断是否空字符串
func IsSpaceOrEmpty(text string) bool {
	count := len(text)
	if count == 0 {
		return true
	}

	for i := 0; i < count; i++ {
		if text[i] != ' ' {
			return false
		}
	}
	return true
}

// RemoveSliceElement 剔除切片元素
func RemoveSliceElement(a []string, b string) []string {
	ret := make([]string, 0, len(a))
	for _, val := range a {
		if val != b {
			ret = append(ret, val)
		}
	}
	return ret
}

// DistinctStringSlice 切片去重
func DistinctStringSlice(strList []string) []string {
	distinctMap := map[string]bool{}
	var distinctList []string

	for _, str := range strList {
		if distinctMap[str] {
			continue
		} else {
			distinctMap[str] = true
			distinctList = append(distinctList, str)
		}
	}
	return distinctList
}

// InStringSlice in str array
func InStringSlice(strFind string, strList []string) bool {
	flag := false
	for _, str := range strList {
		if str == strFind {
			flag = true
			break
		}
	}
	return flag
}

// GetCurrentDir 获取当前应用程序所在目录
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

// CreateDirIfNotExist check exist or create it if not exist
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

//CreateFileIfNotExist check file exist or create it if not exist
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

// StringToBytes converts string to byte slice without a memory allocation.
func StringToBytes(s string) (b []byte) {
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return b
}

// BytesToString converts byte slice to string without a memory allocation.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
