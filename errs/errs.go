package errs

import (
	"fmt"
	"runtime"
)

var traceable = false

// Success 成功提示字符串
const (
	SUCCESS         = "Success"
	AUTH_ERROR      = 995 // 权限错误
	PARAMETER_ERROR = 996 // 参数错误
	SERVER_ERROR    = 997 // 系统错误
	NOT_FOUND       = 998 // 404错误
	UNKNOWN_ERROR   = 999 // 未知错误
)

// SetTraceable 控制error是否带堆栈跟踪
func SetTraceable(x bool) {
	traceable = x
}

func callers() []uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	st := pcs[0:n]
	return st
}

// Error 错误码结构 包含 错误码类型 错误码 错误信息
type Error struct {
	Code int
	Msg  string
	Desc string

	st []uintptr // 调用栈
}

// Error 实现error接口，返回error描述
func (e *Error) Error() string {
	if e == nil {
		return SUCCESS
	}
	return fmt.Sprintf("code:%d, msg:%s", e.Code, e.Msg)
}

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New(code int, msg string) error {
	err := &Error{
		Code: code,
		Msg:  msg,
	}
	if traceable {
		err.st = callers()
	}
	return err
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf(code int, format string, params ...interface{}) error {
	msg := fmt.Sprintf(format, params...)
	return New(code, msg)
}

// Code 通过error获取error code
func Code(e error) int {
	if e == nil {
		return 0
	}
	err, ok := e.(*Error)
	if !ok {
		return UNKNOWN_ERROR
	}
	if err == (*Error)(nil) {
		return 0
	}
	return int(err.Code)
}

// Msg 通过error获取error msg
func Msg(e error) string {
	if e == nil {
		return SUCCESS
	}
	err, ok := e.(*Error)
	if !ok {
		return e.Error()
	}
	if err == (*Error)(nil) {
		return SUCCESS
	}
	return err.Msg
}
