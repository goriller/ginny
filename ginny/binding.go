package ginny

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
)

// Query
func Query(ctx *gin.Context, ptr interface{}) error {
	kind := reflect.TypeOf(ptr)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBindQuery(ptr)
}

// PathVariable
func PathVariable(ctx *gin.Context, ptr interface{}) error {
	kind := reflect.TypeOf(ptr)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBindUri(ptr)
}

// Param
func Param(ctx *gin.Context, ptr interface{}) error {
	kind := reflect.TypeOf(ptr)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	return ctx.ShouldBind(ptr)
}
