package ginny

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ParseJSON 解析请求JSON
func ParseJSON(ctx *gin.Context, obj interface{}) error {
	kind := reflect.TypeOf(obj)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBindJSON(obj)
}

// ParseQuery 解析Query参数
func ParseQuery(ctx *gin.Context, obj interface{}) error {
	kind := reflect.TypeOf(obj)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBindQuery(obj)
}

// ParseForm 解析Form请求
func ParseForm(ctx *gin.Context, obj interface{}) error {
	kind := reflect.TypeOf(obj)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBindWith(obj, binding.Form)
}
