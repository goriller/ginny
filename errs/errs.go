package errs

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New(code codes.Code, msg string) error {
	return status.New(code, msg).Err()
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf(code codes.Code, format string, params ...interface{}) error {
	return status.Newf(code, format, params...).Err()
}
