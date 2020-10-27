package binding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"git.code.oa.com/Ginny/ginny/ginny"

	"github.com/gin-gonic/gin"
)

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// interface{} as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

// defaultMemory
const defaultMemory = 32 << 20

// Form
func Form(ctx *gin.Context, ptr interface{}) error {
	err := shouldBindForm(ctx, ptr)
	if err != nil {
		return err
	}
	return ginny.Validate(ptr)
}

// shouldBindForm
func shouldBindForm(ctx *gin.Context, ptr interface{}) error {
	req := ctx.Request
	h := ctx.GetHeader("content-type")
	if h == "application/json" {
		if req == nil || req.Body == nil {
			return fmt.Errorf("invalid request")
		}
		return decodeJSON(req.Body, ptr)
	} else {
		if err := req.ParseForm(); err != nil {
			return err
		}
		if err := req.ParseMultipartForm(defaultMemory); err != nil {
			if err != http.ErrNotMultipart {
				return err
			}
		}
		if err := mappingByPtr(ptr, req.PostForm, "json"); err != nil {
			return err
		}
	}
	return nil
}

// decodeJSON
func decodeJSON(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return nil
}
