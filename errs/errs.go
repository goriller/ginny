package errs

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New[T int32 | codes.Code](code T, msg string) error {
	return status.New(codes.Code(code), msg).Err()
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf[T int32 | codes.Code](code T, format string, params ...interface{}) error {
	return status.Newf(codes.Code(code), format, params...).Err()
}
