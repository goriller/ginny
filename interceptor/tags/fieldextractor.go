// Copyright (c) The go-grpc-middleware Authors.
// Licensed under the Apache License 2.0.

package tags

import (
	"fmt"
	"reflect"
)

// RequestFieldExtractorFunc is a user-provided function that extracts field information from a gRPC request.
// It is called from tags middleware on arrival of unary request or a server-stream request.
// Keys and values will be added to the context tags of the request. If there are no fields, you should return a nil.
type RequestFieldExtractorFunc func(fullMethod string, req interface{}) map[string]string

type requestFieldsExtractor interface {
	// ExtractRequestFields is a method declared on a Protobuf message that extracts fields from the interface.
	// The values from the extracted fields should be set in the appendToMap, in order to avoid allocations.
	ExtractRequestFields(appendToMap map[string]string)
}

// CodeGenRequestFieldExtractor is a function that relies on code-generated functions that export log fields from requests.
// These are usually coming from a protoc-plugin that generates additional information based on custom field options.
func CodeGenRequestFieldExtractor(_ string, req interface{}) map[string]string {
	if ext, ok := req.(requestFieldsExtractor); ok {
		retMap := make(map[string]string)
		ext.ExtractRequestFields(retMap)
		if len(retMap) == 0 {
			return nil
		}
		return retMap
	}
	return nil
}

// TagBasedRequestFieldExtractor is a function that relies on Go struct tags to export log fields from requests.
// TODO(bwplotka): Add tests/examples https://github.com/grpc-ecosystem/go-grpc-middleware/issues/382
// The tagName is configurable using the tagName variable. Here it would be "log_field".
func TagBasedRequestFieldExtractor(tagName string) RequestFieldExtractorFunc {
	return func(fullMethod string, req interface{}) map[string]string {
		retMap := make(map[string]string)
		reflectMessageTags(req, retMap, tagName)
		if len(retMap) == 0 {
			return nil
		}
		return retMap
	}
}

func reflectMessageTags(msg interface{}, existingMap map[string]string, tagName string) {
	v := reflect.ValueOf(msg)
	// Only deal with pointers to structs.
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return
	}
	// Deref the pointer get to the struct.
	v = v.Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		kind := field.Kind()
		// Only recurse down direct pointers or interfaces, which should only be to nested structs.
		if (kind == reflect.Ptr || kind == reflect.Interface) && field.CanInterface() {
			reflectMessageTags(field.Interface(), existingMap, tagName)
		}
		// In case of arrays/slices (repeated fields) go down to the concrete type.
		if kind == reflect.Array || kind == reflect.Slice {
			if field.Len() == 0 {
				continue
			}
			kind = field.Index(0).Kind()
		}
		// Only be interested in those fields.
		if (kind >= reflect.Bool && kind <= reflect.Float64) || kind == reflect.String {
			if tag := t.Field(i).Tag.Get(tagName); tag != "" {
				existingMap[tag] = fmt.Sprintf("%v", field.Interface())
			}
		}
	}
}
