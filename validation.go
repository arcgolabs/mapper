package mapper

import (
	"fmt"
	"reflect"
)

func (ctx *mappingContext) validate(dstVal reflect.Value) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("mapper: destination validation panicked: %v", recovered)
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
		return fmt.Errorf("mapper: destination validation failed: %w", err)
	}
	return nil
}
