package binding

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	util "github.com/gorillazer/ginny-util"
)

// Query
func Query(ctx *gin.Context, ptr interface{}) error {
	if util.KindOfData(ptr) != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	if err := ctx.ShouldBindQuery(ptr); err != nil {
		return err
	}
	return nil
}

// PathVariable
func PathVariable(ctx *gin.Context, ptr interface{}) error {
	if util.KindOfData(ptr) != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	if err := ctx.ShouldBindUri(ptr); err != nil {
		return err
	}
	return nil
}

// Param
func Param(ctx *gin.Context, ptr interface{}) error {
	if util.KindOfData(ptr) != reflect.Ptr {
		return fmt.Errorf("invalid type, must be pointer")
	}
	if err := ctx.ShouldBind(ptr); err != nil {
		return err
	}
	return nil
}
