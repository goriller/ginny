package ginny

type Code int

const (
	// 错误码值定义
	success       Code = 0
	failed        Code = 1
	paramsError   Code = 400
	accessDenied  Code = 403
	notFound      Code = 404
	internalError Code = 500
	serverTimeout Code = 504
)

type MsgMap map[Code]string

var msgMap = MsgMap{
	success: "success",
	//failed:  "failed",
	//paramsError:   "invalid params",
	//accessDenied:  "access denied",
	//notFound:      "not found",
	//internalError: "internal error",
	//serverTimeout: "server timeout",
}
