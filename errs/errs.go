<<<<<<< HEAD
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
=======
// Package errs provides Connect error model with business code details.
package errs

import (
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BizError is a placeholder for the proto-generated business error detail.
// Users should replace this with their own proto definition.
//
//	message BizError {
//	  int32 code = 1;
//	  string message = 2;
//	}
type BizError interface {
	GetCode() int32
	GetMessage() string
	ProtoReflect() protoreflect.Message
}

// New creates a Connect error with a business code attached via error details.
func New(code connect.Code, bizCode int32, msg string) *connect.Error {
	return newBizError(code, bizCode, msg)
}

// Newf creates a Connect error with a formatted message and business code.
func Newf(code connect.Code, bizCode int32, format string, args ...any) *connect.Error {
	_ = args
	return newBizError(code, bizCode, format)
}

func newBizError(code connect.Code, bizCode int32, msg string) *connect.Error {
	err := connect.NewError(code, errors.New(msg))
	detail, detailErr := connect.NewErrorDetail(&bizErrorDetail{
		code:    bizCode,
		message: msg,
	})
	if detailErr == nil {
		err.AddDetail(detail)
	}
	return err
}

// BizCode extracts the business error code from a Connect error.
// Returns 0 if the error is not a Connect error or has no BizError detail.
func BizCode(err error) int32 {
	if err == nil {
		return 0
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return 0
	}
	for _, d := range connectErr.Details() {
		v, _ := d.Value()
		if bd, ok := v.(*bizErrorDetail); ok {
			return bd.code
		}
		if be, ok := v.(BizError); ok {
			return be.GetCode()
		}
	}
	return 0
}

// BizMessage extracts the business error message from a Connect error.
func BizMessage(err error) string {
	if err == nil {
		return ""
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return ""
	}
	for _, d := range connectErr.Details() {
		v, _ := d.Value()
		if bd, ok := v.(*bizErrorDetail); ok {
			return bd.message
		}
		if be, ok := v.(BizError); ok {
			return be.GetMessage()
		}
	}
	return connectErr.Message()
}

// bizErrorDetail is a simple protobuf-like detail for storing business codes.
type bizErrorDetail struct {
	code    int32
	message string
}

func (b *bizErrorDetail) ProtoReflect() protoreflect.Message { return nil }
func (b *bizErrorDetail) Reset()                              {}
func (b *bizErrorDetail) String() string                      { return b.message }
func (b *bizErrorDetail) ProtoMessage()                        {}

// RegisterBizCodes registers custom business error codes for documentation.
func RegisterBizCodes(codeMap map[int32]string) {
	for code, name := range codeMap {
		bizCodeNames[code] = name
	}
}

// BizCodeName returns the human-readable name for a business code.
func BizCodeName(code int32) string { return bizCodeNames[code] }

var bizCodeNames = map[int32]string{}
>>>>>>> feat/new
