package ginny

import (
	"git.code.oa.com/Ginny/ginny/binding"
	"github.com/gin-gonic/gin"
)

// BindForm
func BindForm(ctx *gin.Context, ptr interface{}) error {
	data := binding.Form(ctx, ptr)
	return binding.Validate(data)
}

// BindQuery
func BindQuery(ctx *gin.Context, ptr interface{}) error {
	data := binding.Query(ctx, ptr)
	return binding.Validate(data)
}

// BindPathVariable
func BindPathVariable(ctx *gin.Context, ptr interface{}) error {
	data := binding.PathVariable(ctx, ptr)
	return binding.Validate(data)
}

// BindParam
func BindParam(ctx *gin.Context, ptr interface{}) error {
	data := binding.Param(ctx, ptr)
	return binding.Validate(data)
}
