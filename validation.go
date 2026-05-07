package mapper

import (
	"reflect"
)

// ValidationFunc adapts a function to ValidationEngine.
type ValidationFunc func(any) error

// Struct validates v.
func (f ValidationFunc) Struct(v any) error {
	return f(v)
}

func (ctx *mappingContext) validate(dstVal reflect.Value) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &ValidationError{ValueType: valueType(dstVal), Cause: panicError{value: recovered}}
		}
	}()

	if ctx == nil || ctx.validator == nil {
		return nil
	}

	var value reflect.Value
	switch dstVal.Kind() {
	case reflect.Pointer:
		if dstVal.IsNil() {
			return nil
		}
		value = dstVal.Elem()
	default:
		value = dstVal
	}

	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
	default:
		return nil
	}

	if err = ctx.validator.Struct(dstVal.Interface()); err != nil {
		return &ValidationError{ValueType: value.Type(), Cause: err}
	}
	return nil
}

func valueType(v reflect.Value) reflect.Type {
	if !v.IsValid() {
		return nil
	}
	return v.Type()
}
