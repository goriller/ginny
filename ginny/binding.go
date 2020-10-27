package ginny

import (
	"git.code.oa.com/Ginny/ginny/binding"
	"github.com/gin-gonic/gin"
)

// BindForm
func BindForm(ctx *gin.Context, ptr interface{}) error {
	return binding.Form(ctx, ptr)
}

// BindQuery
func BindQuery(ctx *gin.Context, ptr interface{}) error {
	return binding.Query(ctx, ptr)
}

// BindPathVariable
func BindPathVariable(ctx *gin.Context, ptr interface{}) error {
	return binding.PathVariable(ctx, ptr)
}

// BindParam
func BindParam(ctx *gin.Context, ptr interface{}) error {
	return binding.Param(ctx, ptr)
}
