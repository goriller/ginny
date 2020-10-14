package ginny

type ErrCode int

const (
	// 错误码值定义
	success       ErrCode = 0
	failed        ErrCode = 1
	paramsError   ErrCode = 400
	accessDenied  ErrCode = 403
	notFound      ErrCode = 404
	internalError ErrCode = 500
	serverTimeout ErrCode = 504
)

type ErrMsgMap map[ErrCode]string

var msgMap = ErrMsgMap{
	success: "success",
	//failed:  "failed",
	//paramsError:   "invalid params",
	//accessDenied:  "access denied",
	//notFound:      "not found",
	//internalError: "internal error",
	//serverTimeout: "server timeout",
}
