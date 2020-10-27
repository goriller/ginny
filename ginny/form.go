package ginny

import (
	"net/http"
	"strings"

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
	h := ctx.GetHeader("content-type")
	if h == "application/x-www-form-urlencoded" {
		if err := ctx.Request.ParseForm(); err != nil {
			return err
		}
		if err := mappingByPtr(ptr, ctx.Request.PostForm, "json"); err != nil {
			return err
		}
	} else if strings.Contains(h, "multipart/form-data") {
		if err := ctx.Request.ParseMultipartForm(defaultMemory); err != nil {
			if err != http.ErrNotMultipart {
				return err
			}
		}
		if err := mappingByPtr(ptr, ctx.Request.Form, "json"); err != nil {
			return err
		}
	} else {
		return ctx.BindJSON(ptr)
	}
	return nil
}
