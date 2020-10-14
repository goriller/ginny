package ginny

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

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
		if err := validate.Struct(dto); err != nil {
			for _, e := range err.(validator.ValidationErrors) {
				err = fmt.Errorf("%s is invalid", e.Field())
			}
			return err
		}
	}

	return nil
}
