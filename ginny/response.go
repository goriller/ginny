package ginny

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// H hash map type
type H map[string]interface{}

// responseResult Common reponse
type responseResult struct {
	Status  int         `json:"-"` //HTTP Status
	Code    string      `json:"code" format:"int"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` //`json:"data,omitempty"`不忽略字段输出null,方便调用方判断
	Err     error       `json:"-"`    //错误
}

// codeMap
var codeMap map[string]string

// SetCodeMap set response code map
func SetCodeMap(m map[string]string) {
	codeMap = m
}

// Error get error
func (r *responseResult) Error() string {
	if r.Message != "" {
		return r.Message
	}
	if r.Err != nil {
		str := r.Err.Error()
		if codeMap != nil {
			if r.Code == "" {
				r.Code = str
			}
			return r.Msg()
		}
		return str
	}
	return ""
}

// Msg automatic matching message
func (r *responseResult) Msg() string {
	if r.Message != "" {
		return r.Message
	}
	if codeMap != nil {
		// matching custom message
		if str, ok := codeMap[r.Code]; ok {
			return str
		}
	}
	// matching common message
	if str, ok := msgMap[r.Code]; ok {
		return str
	}
	return r.Message
}

// response API standard output
func response(ctx *gin.Context, r *responseResult) {
	if r.Err == nil {
		if r.Status == 0 {
			r.Status = http.StatusOK
		}
		if r.Code == "" {
			r.Code = success
		}
		r.Message = r.Msg()

	} else {
		r.Data = nil
		if r.Status == 0 {
			r.Status = http.StatusInternalServerError
		}
		if r.Message == "" {
			r.Message = r.Error()
		}
		if r.Code == "" {
			r.Code = failed
		}
	}
	ctx.JSON(r.Status, r)
}

// ResponseSuccess
func ResponseSuccess(ctx *gin.Context, data interface{}) {
	resp := &responseResult{
		Status: http.StatusOK,
		Code:   success,
		Data:   data,
	}

	response(ctx, resp)
}

// ResponseError
//
// options: [code int, message string, status int]
func ResponseError(ctx *gin.Context, err error, options ...interface{}) {
	resp := &responseResult{
		Status: http.StatusOK,
		// Code:   failed,
		Err: err,
	}
	pickOptions(resp, options)
	response(ctx, resp)
}

// pickOptions
func pickOptions(resp *responseResult, options []interface{}) {
	lens := len(options)
	if lens == 0 {
		return
	}
	if lens > 2 {
		if c, ok := options[0].(string); ok {
			resp.Code = c
		}
		if c, ok := options[1].(string); ok {
			resp.Message = c
		}
		if c, ok := options[2].(int); ok {
			if c >= 100 && c <= 510 {
				resp.Status = c
			}
		}
	} else if lens == 2 {
		if c, ok := options[0].(string); ok {
			resp.Code = c
		}
		if c, ok := options[1].(string); ok {
			resp.Message = c
		}
	} else {
		if c, ok := options[0].(string); ok {
			resp.Code = c
		}
	}
}
