package ginny

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HTTPResponseResult Common reponse
type HTTPResponseResult struct {
	Status  int         `json:"-"` //HTTP Status
	Code    Code        `json:"code" format:"int"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //`json:"data,omitempty"`不忽略字段输出null,方便调用方判断
	Err     error       `json:"-"`    //错误

	codeMap map[Code]string `json:"-"` //项目自定义错误信息
}

// Error get error
func (r *HTTPResponseResult) Error() string {
	if r.Err != nil {
		return r.Err.Error()
	}
	return r.Message
}

// Msg automatic matching message
func (r *HTTPResponseResult) Msg() string {
	if r.Code == 0 || r.Message != "" {
		return r.Message
	}
	if r.codeMap != nil {
		// matching custom message
		if str, ok := r.codeMap[r.Code]; ok {
			return str
		}
	}
	// matching common message
	if str, ok := Msg[r.Code]; ok {
		return str
	}
	return r.Message
}

// response API standard output
func response(ctx *gin.Context, r *HTTPResponseResult) {
	if r.Message == "" {
		r.Message = r.Msg()
	}
	if r.Err == nil {
		if r.Status == 0 {
			r.Status = http.StatusOK
		}
		if r.Code == 0 {
			r.Code = Success
		}
	} else {
		r.Data = nil
		if r.Status == 0 {
			r.Status = http.StatusInternalServerError
		}
		if r.Message == "" {
			r.Message = r.Error()
		}
		if r.Code == 0 {
			r.Code = Failed
		}
		// error log
		//log.ErrorContext(ctx, r.Message,
		//	zap.Error(r.Err),
		//)
	}
	ctx.JSON(r.Status, r)
	return
}

// ResponseSuccess
func ResponseSuccess(ctx *gin.Context, data interface{}) {
	resp := NewResultBuilder()
	resp.Status(http.StatusOK).Code(Success)
	resp.Data(data).Response(ctx)
	return
}

// ResponseError
//
// options: [code Code, message string, status int]
func ResponseError(ctx *gin.Context, err error, options ...interface{}) {
	resp := NewResultBuilder()
	resp.Status(http.StatusOK).Code(Failed)
	pickOptions(resp, options)
	resp.Error(err).Response(ctx)
	return
}

// ParamErrorResponse
func ResponseParamError(ctx *gin.Context, err error) {
	resp := NewResultBuilder()
	resp.Status(http.StatusBadRequest).Code(ParamsError)
	resp.Error(err).Response(ctx)
	return
}

// ResponseAccessDenied
func ResponseAccessDenied(ctx *gin.Context, err error) {
	resp := NewResultBuilder()
	resp.Status(http.StatusForbidden).Code(AccessDenied)
	resp.Error(err).Response(ctx)
	return
}

// ResponseNotFound
func ResponseNotFound(ctx *gin.Context, err error) {
	resp := NewResultBuilder()
	resp.Status(http.StatusNotFound).Code(NotFound)
	resp.Error(err).Response(ctx)
	return
}

// ResponseInternalError
func ResponseInternalError(ctx *gin.Context, err error) {
	resp := NewResultBuilder()
	resp.Status(http.StatusInternalServerError).Code(InternalError)
	resp.Error(err).Response(ctx)
	return
}

// ResponseServerTimeout
func ResponseServerTimeout(ctx *gin.Context, err error) {
	resp := NewResultBuilder()
	resp.Status(http.StatusGatewayTimeout).Code(ServerTimeout)
	resp.Error(err).Response(ctx)
	return
}

// pickMsg
func pickMsg(resp *ResultBuilder, messages ...string) {
	if len(messages) > 0 {
		resp.Message(messages[0])
	}
	return
}

// pickOptions
func pickOptions(resp *ResultBuilder, options []interface{}) {
	for i := 0; i < len(options); i++ {
		if c, ok := options[i].(int); ok {
			if c < 100 || c > 510 {
				continue
			}
			resp.Status(c)
		} else if c, ok := options[i].(Code); ok {
			resp.Code(c)
		} else if c, ok := options[i].(string); ok {
			resp.Message(c)
		}
	}
	return
}

// ResultBuilder builder pattern code
type ResultBuilder struct {
	result *HTTPResponseResult
}

// NewResultBuilder get instances of ResultBuilder
func NewResultBuilder(m ...map[Code]string) *ResultBuilder {
	result := &HTTPResponseResult{
		codeMap: nil,
	}
	if m != nil && m[0] != nil {
		result.codeMap = m[0]
	}
	b := &ResultBuilder{result: result}
	return b
}

// Code setter
func (b *ResultBuilder) Code(code Code) *ResultBuilder {
	b.result.Code = code
	return b
}

// Message setter
func (b *ResultBuilder) Message(message string) *ResultBuilder {
	b.result.Message = message
	return b
}

// Data setter
func (b *ResultBuilder) Data(data interface{}) *ResultBuilder {
	b.result.Data = data
	return b
}

// Err setter
func (b *ResultBuilder) Error(err error) *ResultBuilder {
	b.result.Err = err
	return b
}

// Status setter
func (b *ResultBuilder) Status(status int) *ResultBuilder {
	b.result.Status = status
	return b
}

// Build
func (b *ResultBuilder) Build() *HTTPResponseResult {
	return b.result
}

// Response
func (b *ResultBuilder) Response(ctx *gin.Context) {
	response(ctx, b.result)
	return
}
