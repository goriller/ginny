package ginny

import (
	"github.com/gin-gonic/gin/binding"
)

// Validator interface
type Validator interface {
	Validate() error
}

// Validate for struct: Validate(&dto)
func Validate(dto interface{}) error {
	//kind := reflect.TypeOf(dto)
	//if kind.Kind() != reflect.Ptr {
	//	return fmt.Errorf("invalid dto type, must be pointer")
	//}

	// If struct implements the Validator interface, call it
	if v, ok := dto.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	} else {
		// Perform validate.Struct verification by default
		if binding.Validator == nil {
			return nil
		}
		return binding.Validator.ValidateStruct(dto)
	}

	return nil
}
