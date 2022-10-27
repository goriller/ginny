package errs

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New(code any, msg string) error {
	switch c := code.(type) {
	case codes.Code:
		return status.New(c, msg).Err()
	default:
		return status.New(codes.Code(c.(int32)), msg).Err()
	}
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf(code any, format string, params ...interface{}) error {
	switch c := code.(type) {
	case codes.Code:
		return status.Newf(c, format, params...).Err()
	default:
		return status.Newf(codes.Code(c.(int32)), format, params...).Err()
	}
}
