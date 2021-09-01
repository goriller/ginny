package res

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorillazer/ginny/errs"
)

// HandlerFunc
type HandlerFunc func(c *gin.Context) (*Response, error)

// Response 返回结构
type Response struct {
	Status   int         `json:"-"`
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Request  string      `json:"request"`
	Response interface{} `json:"response,omitempty"`
}

// 实现接口
func (r *Response) Error() string {
	return r.Message
}

// Responser
type Responser func(r *Response)

// WithStatus
func WithStatus(s int) Responser {
	return func(r *Response) {
		if s != 0 {
			r.Status = s
		}
	}
}

// WithCode
func WithCode(c int) Responser {
	return func(r *Response) {
		if c != 0 {
			r.Code = c
		}
	}
}

// WithMessage
func WithMessage(msg string) Responser {
	return func(r *Response) {
		if msg != "" {
			r.Message = msg
		}
	}
}

// NewResponse
func NewResponse(data interface{}, rsp ...Responser) *Response {
	r := &Response{
		Response: data,
	}
	for _, v := range rsp {
		v(r)
	}
	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	if r.Message == "" && r.Status == http.StatusOK {
		r.Message = errs.SUCCESS
	}
	return r
}

// Wrapper
func Wrapper(handler HandlerFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		var resp *Response
		r, err := handler(c)
		if err != nil {
			switch v := err.(type) {
			case *errs.Error:
				resp = BusinessError(v)
			case error:
				resp = UnknownError(v)
			default:
				resp = ServerError()
			}
		} else {
			resp = r
		}
		resp.Request = c.Request.URL.String()
		c.JSON(resp.Status, resp)
	}
}

// Success
func Success(data interface{}) *Response {
	return NewResponse(data,
		WithStatus(http.StatusOK),
		WithMessage(errs.SUCCESS),
		WithCode(0),
	)
}

// Fail
func Fail(e *errs.Error, status ...int) *Response {
	s := http.StatusInternalServerError
	if len(status) > 0 {
		s = status[0]
	}
	return NewResponse(nil,
		WithStatus(s),
		WithCode(e.Code),
		WithMessage(e.Msg),
	)
}

// 未知错误
func UnknownError(e error) *Response {
	return NewResponse(nil,
		WithStatus(http.StatusInternalServerError),
		WithCode(errs.UNKNOWN_ERROR),
		WithMessage(e.Error()),
	)
}

// 业务错误
func BusinessError(e *errs.Error) *Response {
	return NewResponse(nil,
		WithStatus(http.StatusInternalServerError),
		WithCode(e.Code),
		WithMessage(e.Msg),
	)
}

// 参数错误
func ParameterError(message string) *Response {
	return NewResponse(nil,
		WithStatus(http.StatusBadRequest),
		WithCode(errs.PARAMETER_ERROR),
		WithMessage(message),
	)
}

// 500 错误处理
func ServerError() *Response {
	return NewResponse(nil,
		WithStatus(http.StatusInternalServerError),
		WithCode(errs.SERVER_ERROR),
		WithMessage(http.StatusText(http.StatusInternalServerError)),
	)
}
