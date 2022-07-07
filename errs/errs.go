package errs

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New(code uint32, msg string) error {
	return status.New(codes.Code(code), msg).Err()
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf(code uint32, format string, params ...interface{}) error {
	return status.Newf(codes.Code(code), format, params...).Err()
}
