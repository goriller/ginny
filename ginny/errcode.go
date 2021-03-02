package ginny

type (
	ErrCode   int
	ErrMsgMap map[string]string
)

const (
	// 错误码值定义
	success       = "0"
	failed        = "1"
	paramsError   = "400"
	accessDenied  = "403"
	notFound      = "404"
	internalError = "500"
	serverTimeout = "504"
)

var msgMap = ErrMsgMap{
	success:       "success",
	failed:        "failed",
	paramsError:   "invalid params",
	accessDenied:  "access denied",
	notFound:      "not found",
	internalError: "internal error",
	serverTimeout: "server timeout",
}
