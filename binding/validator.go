package binding

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

// Validator
type Validator interface {
	Validate() error
}

// RegisterValidation Custom verification
//
// func checkMobile(fl validator.FieldLevel) bool {
//    mobile := strconv.Itoa(int(fl.Field().Uint()))
//    re := `^1[3456789]\d{9}$`
//    r := regexp.MustCompile(re)
//    return r.MatchString(mobile)
//}
func RegisterValidation(fn []validator.Func) error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		for _, f := range fn {
			if err := v.RegisterValidation(getFunctionName(f, '/', '.'), f); err != nil {
				return err
			}
		}
	}
	return nil
}

// Valid
func Valid(v interface{}) error {
	// validate
	if vv, ok := v.(Validator); ok {
		if err := vv.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// GetFunctionName 获取函数名称
func getFunctionName(i interface{}, seps ...rune) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// 用 seps 进行分割
	fields := strings.FieldsFunc(fn, func(sep rune) bool {
		for _, s := range seps {
			if sep == s {
				return true
			}
		}
		return false
	})

	if size := len(fields); size > 0 {
		return fields[size-1]
	}
	return ""
}
