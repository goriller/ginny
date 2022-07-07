// Package logging implements grpc logging middleware.
package logging

import (
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func clearMessageBytes(m pref.Message) {
	m.Range(func(fd pref.FieldDescriptor, val pref.Value) bool {
		clearValueBytes(m, val, fd)
		return true
	})
}

func clearValueBytes(m pref.Message, val pref.Value, fd pref.FieldDescriptor) {
	switch {
	case fd.IsList():
		list := val.List()
		for i := 0; i < list.Len(); i++ {
			item := list.Get(i)
			clearMessageBytes(item.Message())
		}
	case fd.IsMap():
		mmap := val.Map()
		mmap.Range(func(k pref.MapKey, v pref.Value) bool {
			clearSingularBytes(m, v, fd.MapValue())
			return true
		})
	default:
		clearSingularBytes(m, val, fd)
	}
}

func clearSingularBytes(m pref.Message, val pref.Value, fd pref.FieldDescriptor) {
	if !val.IsValid() {
		return
	}
	switch kind := fd.Kind(); kind {
	case pref.BytesKind:
		m.Clear(fd)
	case pref.MessageKind, pref.GroupKind:
		clearMessageBytes(val.Message())
	}
}
