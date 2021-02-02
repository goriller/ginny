package ginny

import (
	"git.code.oa.com/Ginny/ginny/binding"
	"github.com/gin-gonic/gin"
)

// BindQuery
func BindQuery(ctx *gin.Context, ptr interface{}) error {
	data := ctx.ShouldBindQuery(ptr)
	return binding.Validate(data)
}

// BindPathVariable
func BindPathVariable(ctx *gin.Context, ptr interface{}) error {
	data := ctx.ShouldBindUri(ptr)
	return binding.Validate(data)
}

// BindParam
func BindParam(ctx *gin.Context, ptr interface{}) error {
	data := ctx.ShouldBind(ptr)
	return binding.Validate(data)
}
