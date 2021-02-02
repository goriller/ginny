package ginny

import (
	"github.com/gin-gonic/gin"
)

// Query
func Query(ctx *gin.Context, ptr interface{}) error {
	return ctx.ShouldBindQuery(ptr)
}

// PathVariable
func PathVariable(ctx *gin.Context, ptr interface{}) error {
	return ctx.ShouldBindUri(ptr)
}

// Param
func Param(ctx *gin.Context, ptr interface{}) error {
	return ctx.ShouldBind(ptr)
}
