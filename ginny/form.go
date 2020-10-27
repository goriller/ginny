package ginny

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// defaultMemory
const defaultMemory = 32 << 20

// Post
func Post(ctx *gin.Context, ptr interface{}) error {
	return shouldBindForm(ctx, ptr)
}

// shouldBindForm
func shouldBindForm(ctx *gin.Context, ptr interface{}) error {
	if err := ctx.Request.ParseForm(); err != nil {
		return err
	}
	if err := ctx.Request.ParseMultipartForm(defaultMemory); err != nil {
		if err != http.ErrNotMultipart {
			return err
		}
	}
	if err := mappingByPtr(ptr, ctx.Request.Form, "json"); err != nil {
		return err
	}
	return nil
}
