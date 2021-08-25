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
	if err := ctx.ShouldBindQuery(ptr); err != nil {
		return err
	}
	return Valid(ptr)
}

// PathVariable
func PathVariable(ctx *gin.Context, ptr interface{}) error {
	kind := reflect.TypeOf(ptr)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	if err := ctx.ShouldBindUri(ptr); err != nil {
		return err
	}
	return Valid(ptr)
}

// Param
func Param(ctx *gin.Context, ptr interface{}) error {
	kind := reflect.TypeOf(ptr)
	if kind.Kind() != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	if err := ctx.ShouldBind(ptr); err != nil {
		return err
	}
	return Valid(ptr)
}
