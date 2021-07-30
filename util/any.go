package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrConvert  = errors.New("convert error")
	ErrNotFound = errors.New("value not found")
	ErrIndex    = errors.New("index out of range")
)

// Any is a converter to transfer data types
type Any struct {
	value interface{} // the real value
}

// ToAny convert any value to type Any
func ToAny(v interface{}) Any {
	if a, ok := v.(Any); ok {
		return a
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.Elem().CanInterface() {
		v = rv.Elem().Interface()
	}
	return Any{value: v}
}

// Bool convert the value to a Bool value or fallback if error
func (any Any) Bool(fallback ...bool) bool {
	v, e := any.TryBool()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Float32 convert the value to a Float32 value or fallback if error
func (any Any) Float32(fallback ...float32) float32 {
	v, e := any.TryFloat32()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Float64 convert the value to a Float64 value or fallback if error
func (any Any) Float64(fallback ...float64) float64 {
	v, e := any.TryFloat64()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Int convert the value to a Int value or fallback if error
func (any Any) Int(fallback ...int) int {
	v, e := any.TryInt()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Int8 convert the value to a Int8 value or fallback if error
func (any Any) Int8(fallback ...int8) int8 {
	v, e := any.TryInt8()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Int16 convert the value to a Int16 value or fallback if error
func (any Any) Int16(fallback ...int16) int16 {
	v, e := any.TryInt16()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Int32 convert the value to a Int32 value or fallback if error
func (any Any) Int32(fallback ...int32) int32 {
	v, e := any.TryInt32()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Int64 convert the value to a Int64 value or fallback if error
func (any Any) Int64(fallback ...int64) int64 {
	v, e := any.TryInt64()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Map convert the value to a Map[string]Any value or fallback if error
func (any Any) Map(fallback ...map[string]Any) map[string]Any {
	v, e := any.TryMap()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// MapGet return the value of pattern in the Map Value
func (any Any) MapGet(pattern string, fallback ...Any) Any {
	v, e := any.TryMapGet(pattern)
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Slice convert the value to a AnySlice value or fallback if error
func (any Any) Slice(fallback ...AnySlice) AnySlice {
	v, e := any.TrySlice()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// String convert the value to a string
func (any Any) String() string {
	if v, ok := any.value.(string); ok {
		return v
	}
	return fmt.Sprint(any.value)
}

// Uint convert the value to a Uint value or fallback if error
func (any Any) Uint(fallback ...uint) uint {
	v, e := any.TryUint()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Uint8 convert the value to a Uint8 value or fallback if error
func (any Any) Uint8(fallback ...uint8) uint8 {
	v, e := any.TryUint8()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Uint16 convert the value to a Uint16 value or fallback if error
func (any Any) Uint16(fallback ...uint16) uint16 {
	v, e := any.TryUint16()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Uint32 convert the value to a Uint32 value or fallback if error
func (any Any) Uint32(fallback ...uint32) uint32 {
	v, e := any.TryUint32()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Uint64 convert the value to a Uint64 value or fallback if error
func (any Any) Uint64(fallback ...uint64) uint64 {
	v, e := any.TryUint64()
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Interface return the real value as a interface{}
func (any Any) Interface() interface{} {
	return any.value
}

// IsNil return whether the real value is nil
func (any Any) IsNil() bool {
	return any.value == nil
}

// TrySlice try to convert the value to AnySlice
func (any Any) TrySlice() (AnySlice, error) {
	if v, ok := any.value.(AnySlice); ok {
		return v, nil
	}

	rv := reflect.ValueOf(any.value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.String:
	default:
		return nil, ErrConvert
	}

	val := make(AnySlice, 0)

	for i := 0; i < rv.Len(); i++ {
		iv := rv.Index(i)
		if !iv.CanInterface() {
			return nil, ErrConvert
		}
		val = append(val, ToAny(iv.Interface()))
	}

	return val, nil
}

// TryMap try to convert the value to map[string]Any
func (any Any) TryMap() (map[string]Any, error) {

	val := make(map[string]Any)

	rv := reflect.ValueOf(any.value)
	if rv.Kind() != reflect.Map {
		return nil, ErrConvert
	}

	for _, mk := range rv.MapKeys() {
		mv := rv.MapIndex(mk)
		if !mk.CanInterface() || !mv.CanInterface() {
			return nil, ErrConvert
		}
		val[ToAny(mk.Interface()).String()] = ToAny(mv.Interface())
	}

	return val, nil
}

// TryMapGet try to get value of pattern in the map value
func (any Any) TryMapGet(pattern string) (Any, error) {
	var (
		val Any
		err error
	)
	m, err := any.TryMap()
	if err != nil {
		return val, err
	}
	s := strings.Split(pattern, ".")
	emptyMap := make(map[string]Any)
	for _, k := range s {
		var has bool
		if val, has = m[k]; !has {
			err = ErrNotFound
			return val, err
		}
		m = val.Map(emptyMap)
	}

	return val, nil
}

// TryBool try to convert the value to Bool
func (any Any) TryBool() (bool, error) {
	if v, ok := any.value.(bool); ok {
		return v, nil
	}
	return strconv.ParseBool(any.toString())
}

// TryFloat32 try to convert the value to Float32
func (any Any) TryFloat32() (float32, error) {
	if v, ok := any.value.(float32); ok {
		return v, nil
	}
	v64, err := strconv.ParseFloat(any.toString(), 32)
	return float32(v64), err
}

// TryFloat64 try to convert the value to Float64
func (any Any) TryFloat64() (float64, error) {
	if v, ok := any.value.(float64); ok {
		return v, nil
	}
	v64, err := strconv.ParseFloat(any.toString(), 64)
	return v64, err
}

// TryInt try to convert the value to Int
func (any Any) TryInt() (int, error) {
	if v, ok := any.value.(int); ok {
		return v, nil
	}
	return strconv.Atoi(any.toString())
}

// TryInt8 try to convert the value to Int8
func (any Any) TryInt8() (int8, error) {
	if v, ok := any.value.(int8); ok {
		return v, nil
	}
	v64, err := strconv.ParseInt(any.toString(), 10, 8)
	return int8(v64), err
}

// TryInt16 try to convert the value to Int16
func (any Any) TryInt16() (int16, error) {
	if v, ok := any.value.(int16); ok {
		return v, nil
	}
	v64, err := strconv.ParseInt(any.toString(), 10, 16)
	return int16(v64), err
}

// TryInt32 try to convert the value to Int32
func (any Any) TryInt32() (int32, error) {
	if v, ok := any.value.(int32); ok {
		return v, nil
	}
	v64, err := strconv.ParseInt(any.toString(), 10, 32)
	return int32(v64), err
}

// TryInt64 try to convert the value to Int64
func (any Any) TryInt64() (int64, error) {
	if v, ok := any.value.(int64); ok {
		return v, nil
	}
	v64, err := strconv.ParseInt(any.toString(), 10, 64)
	return v64, err
}

// TryUint try to convert the value to Uint
func (any Any) TryUint() (uint, error) {
	if v, ok := any.value.(uint); ok {
		return v, nil
	}
	v64, err := strconv.ParseUint(any.toString(), 10, 0)
	return uint(v64), err
}

// TryUint8 try to convert the value to Uint8
func (any Any) TryUint8() (uint8, error) {
	if v, ok := any.value.(uint8); ok {
		return v, nil
	}
	v64, err := strconv.ParseUint(any.toString(), 10, 8)
	return uint8(v64), err
}

// TryUint16 try to convert the value to Uint16
func (any Any) TryUint16() (uint16, error) {
	if v, ok := any.value.(uint16); ok {
		return v, nil
	}
	v64, err := strconv.ParseUint(any.toString(), 10, 16)
	return uint16(v64), err
}

// TryUint32 try to convert the value to Uint32
func (any Any) TryUint32() (uint32, error) {
	if v, ok := any.value.(uint32); ok {
		return v, nil
	}
	v64, err := strconv.ParseUint(any.toString(), 10, 32)
	return uint32(v64), err
}

// TryUint64 try to convert the value to Uint64
func (any Any) TryUint64() (uint64, error) {
	if v, ok := any.value.(uint64); ok {
		return v, nil
	}
	v64, err := strconv.ParseUint(any.toString(), 10, 64)
	return v64, err
}

// UnmarshalJSON unmarshal JSON to the value of Any
func (any *Any) UnmarshalJSON(b []byte) (err error) {
	var m interface{}

	if err = json.Unmarshal(b, &m); err != nil {
		return
	}
	any.value = m
	return
}

// MarshalJSON marshal the value of Any to JSON
func (any Any) MarshalJSON() ([]byte, error) {
	return json.Marshal(any.value)
}

func (any Any) toString() string {
	if any.value == nil {
		return ""
	}
	if v, ok := any.value.(string); ok {
		return strings.TrimSpace(v)
	}
	v := fmt.Sprint(any.value)
	return strings.TrimSpace(v)
}

// AnySlice is []Any
type AnySlice []Any

// TryIndex returns the AnySlice's i'th element
func (s AnySlice) TryIndex(i int) (Any, error) {
	if len(s) < i {
		return Any{}, ErrIndex
	}
	return s[i], nil
}

// Index returns the AnySlice's i'th element or fallback if error
func (s AnySlice) Index(i int, fallback ...Any) Any {
	v, e := s.TryIndex(i)
	if e != nil && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Length returns the length of slice
func (s AnySlice) Length() int {
	return len(s)
}

// SliceInterface returns AnySlice as []interface{}
func (s AnySlice) SliceInterface() []interface{} {
	size := s.Length()
	v := make([]interface{}, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Interface()
	}
	return v
}

// SliceBool returns AnySlice as []bool
func (s AnySlice) SliceBool() []bool {
	size := s.Length()
	v := make([]bool, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Bool()
	}
	return v
}

// SliceFloat32 returns AnySlice as []float32
func (s AnySlice) SliceFloat32() []float32 {
	size := s.Length()
	v := make([]float32, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Float32()
	}
	return v
}

// SliceFloat64 returns AnySlice as []float64
func (s AnySlice) SliceFloat64() []float64 {
	size := s.Length()
	v := make([]float64, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Float64()
	}
	return v
}

// SliceInt returns AnySlice as []int
func (s AnySlice) SliceInt() []int {
	size := s.Length()
	v := make([]int, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Int()
	}
	return v
}

// SliceInt8 returns AnySlice as []int8
func (s AnySlice) SliceInt8() []int8 {
	size := s.Length()
	v := make([]int8, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Int8()
	}
	return v
}

// SliceInt16 returns AnySlice as []int16
func (s AnySlice) SliceInt16() []int16 {
	size := s.Length()
	v := make([]int16, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Int16()
	}
	return v
}

// SliceInt32 returns AnySlice as []int32
func (s AnySlice) SliceInt32() []int32 {
	size := s.Length()
	v := make([]int32, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Int32()
	}
	return v
}

// SliceInt64 returns AnySlice as []int64
func (s AnySlice) SliceInt64() []int64 {
	size := s.Length()
	v := make([]int64, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Int64()
	}
	return v
}

// SliceString returns AnySlice as []string
func (s AnySlice) SliceString() []string {
	size := s.Length()
	v := make([]string, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).String()
	}
	return v
}

// SliceUint returns AnySlice as []uint
func (s AnySlice) SliceUint() []uint {
	size := s.Length()
	v := make([]uint, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Uint()
	}
	return v
}

// SliceUint8 returns AnySlice as []uint8
func (s AnySlice) SliceUint8() []uint8 {
	size := s.Length()
	v := make([]uint8, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Uint8()
	}
	return v
}

// SliceUint16 returns AnySlice as []uint16
func (s AnySlice) SliceUint16() []uint16 {
	size := s.Length()
	v := make([]uint16, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Uint16()
	}
	return v
}

// SliceUint32 returns AnySlice as []uint32
func (s AnySlice) SliceUint32() []uint32 {
	size := s.Length()
	v := make([]uint32, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Uint32()
	}
	return v
}

// SliceUint64 returns AnySlice as []uint64
func (s AnySlice) SliceUint64() []uint64 {
	size := s.Length()
	v := make([]uint64, size)
	for i := 0; i < size; i++ {
		v[i] = s.Index(i).Uint64()
	}
	return v
}

// Bool is a shortcut of ToAny(v).Bool()
func Bool(v interface{}) bool { return ToAny(v).Bool() }

// Float32 is a shortcut of ToAny(v).Float32()
func Float32(v interface{}) float32 { return ToAny(v).Float32() }

// Float64 is a shortcut of ToAny(v).Float64()
func Float64(v interface{}) float64 { return ToAny(v).Float64() }

// Int is a shortcut of ToAny(v).Int()
func Int(v interface{}) int { return ToAny(v).Int() }

// Int16 is a shortcut of ToAny(v).Int16()
func Int16(v interface{}) int16 { return ToAny(v).Int16() }

// Int32 is a shortcut of ToAny(v).Int32()
func Int32(v interface{}) int32 { return ToAny(v).Int32() }

// Int64 is a shortcut of ToAny(v).Int64()
func Int64(v interface{}) int64 { return ToAny(v).Int64() }

// Int8 is a shortcut of ToAny(v).Int8()
func Int8(v interface{}) int8 { return ToAny(v).Int8() }

// Map is a shortcut of ToAny(v).Map()
func Map(v interface{}) map[string]Any { return ToAny(v).Map() }

// Slice is a shortcut of ToAny(v).Slice()
func Slice(v interface{}) AnySlice { return ToAny(v).Slice() }

// String is a shortcut of ToAny(v).String()
func String(v interface{}) string { return ToAny(v).String() }

// Uint is a shortcut of ToAny(v).Uint()
func Uint(v interface{}) uint { return ToAny(v).Uint() }

// Uint16 is a shortcut of ToAny(v).Uint16()
func Uint16(v interface{}) uint16 { return ToAny(v).Uint16() }

// Uint32 is a shortcut of ToAny(v).Uint32()
func Uint32(v interface{}) uint32 { return ToAny(v).Uint32() }

// Uint64 is a shortcut of ToAny(v).Uint64()
func Uint64(v interface{}) uint64 { return ToAny(v).Uint64() }

// Uint8 is a shortcut of ToAny(v).Uint8()
func Uint8(v interface{}) uint8 { return ToAny(v).Uint8() }

// BoolPrt return v as a *bool
func BoolPrt(v interface{}) (p *bool) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Bool()
		p = &pv
	}
	return
}

// Float32Prt return v as a *float32
func Float32Prt(v interface{}) (p *float32) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Float32()
		p = &pv
	}
	return
}

// Float64Prt return v as a *float64
func Float64Prt(v interface{}) (p *float64) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Float64()
		p = &pv
	}
	return
}

// IntPrt return v as a *int
func IntPrt(v interface{}) (p *int) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Int()
		p = &pv
	}
	return
}

// Int16Prt return v as a *int16
func Int16Prt(v interface{}) (p *int16) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Int16()
		p = &pv
	}
	return
}

// Int32Prt return v as a *int32
func Int32Prt(v interface{}) (p *int32) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Int32()
		p = &pv
	}
	return
}

// Int64Prt return v as a *int64
func Int64Prt(v interface{}) (p *int64) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Int64()
		p = &pv
	}
	return
}

// Int8Prt return v as a *int8
func Int8Prt(v interface{}) (p *int8) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Int8()
		p = &pv
	}
	return
}

// StringPrt return v as a *string
func StringPrt(v interface{}) (p *string) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.String()
		p = &pv
	}
	return
}

// UintPrt return v as a *uint
func UintPrt(v interface{}) (p *uint) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Uint()
		p = &pv
	}
	return
}

// Uint16Prt return v as a *uint16
func Uint16Prt(v interface{}) (p *uint16) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Uint16()
		p = &pv
	}
	return
}

// Uint32Prt return v as a *uint32
func Uint32Prt(v interface{}) (p *uint32) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Uint32()
		p = &pv
	}
	return
}

// Uint64Prt return v as a *uint64
func Uint64Prt(v interface{}) (p *uint64) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Uint64()
		p = &pv
	}
	return
}

// Uint8Prt return v as a *uint8
func Uint8Prt(v interface{}) (p *uint8) {
	if val := ToAny(v); !val.IsNil() {
		pv := val.Uint8()
		p = &pv
	}
	return
}

// Deduplicate returns a de-duplicated slice, it return same input if the input value is not slice
func Deduplicate(slice interface{}) interface{} {
	value := reflect.ValueOf(slice)
	if value.Kind() != reflect.Slice {
		return slice
	}

	m := make(map[int]reflect.Value)
	n := 0

	for i := 0; i < value.Len(); i++ {
		v := value.Index(i)
		has := false
		for _, val := range m {
			if reflect.DeepEqual(v.Interface(), val.Interface()) {
				has = true
				break
			}
		}
		if has {
			continue
		}

		value.Index(n).Set(v)
		m[n] = v
		n++
	}

	return value.Slice(0, n).Interface()
}

// Contains return true if the target in the slice
// it panic if input slice is not a slice type
func Contains(slice interface{}, target interface{}) bool {
	value := reflect.ValueOf(slice)
	if value.Kind() != reflect.Slice {
		panic("input is not a slice type")
	}

	for i := 0; i < value.Len(); i++ {
		v := value.Index(i)
		if reflect.DeepEqual(v.Interface(), target) {
			return true
		}
	}

	return false
}

// IsEmptyString 判断是否空字符串
func IsEmptyString(text string) bool {
	s := strings.TrimSpace(text)
	return len(s) <= 0
}

// IsSpaceOrEmpty 判断是否空字符串,字符串仅包括空格也返回true
func IsSpaceOrEmpty(text string) bool {
	count := len(text)
	if count == 0 {
		return true
	}

	for i := 0; i < count; i++ {
		if text[i] != ' ' {
			return false
		}
	}
	return true
}

// RemoveSliceElement 剔除切片元素
func RemoveSliceElement(a []string, b string) []string {
	ret := make([]string, 0, len(a))
	for _, val := range a {
		if val != b {
			ret = append(ret, val)
		}
	}
	return ret
}

// DistinctStringSlice 切片去重
func DistinctStringSlice(strList []string) []string {
	distinctMap := map[string]bool{}
	var distinctList []string

	for _, str := range strList {
		if distinctMap[str] {
			continue
		} else {
			distinctMap[str] = true
			distinctList = append(distinctList, str)
		}
	}
	return distinctList
}

// InStringSlice 判断字符串是否存在字符串切片内
func InStringSlice(strFind string, strList []string) bool {
	flag := false
	for _, str := range strList {
		if str == strFind {
			flag = true
			break
		}
	}
	return flag
}
