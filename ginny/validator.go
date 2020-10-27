package ginny

import "git.code.oa.com/Ginny/ginny/binding"

// Validate for struct: Validate(&dto)
func Validate(dto interface{}) error {
	return binding.Validate(dto)
}
