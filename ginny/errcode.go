package ginny

type Code int

const (
	// 错误码值定义
	Success       Code = 0
	Failed        Code = 1
	ParamsError   Code = 400
	AccessDenied  Code = 403
	NotFound      Code = 404
	InternalError Code = 500
	ServerTimeout Code = 504
)

type MsgMap map[Code]string

var Msg = MsgMap{
	Success: "success",
	//Failed:  "failed",
	//ParamsError:   "invalid params",
	//AccessDenied:  "access denied",
	//NotFound:      "not found",
	//InternalError: "internal error",
	//ServerTimeout: "server timeout",
}
