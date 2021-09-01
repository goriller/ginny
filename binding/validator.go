package binding

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	util "github.com/gorillazer/ginny-util"
	"gopkg.in/go-playground/validator.v9"
)

func init() {
	binding.Validator = &DefaultValidator{}
}

// DefaultValidator
type DefaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

// ValidateStruct
func (v *DefaultValidator) ValidateStruct(obj interface{}) error {
	if util.KindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return error(err)
		}
	}
	return nil
}

// ValidateVar
func (v *DefaultValidator) ValidateVar(field interface{}, tag string) error {
	v.lazyinit()
	return v.validate.Var(field, tag)
}

// Engine
func (v *DefaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

func (v *DefaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
		// add any custom validations etc. here
	})
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
