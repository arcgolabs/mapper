package mapper

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

func (ctx *mappingContext) mapDefaultValue(raw string, dstVal reflect.Value, path string) error {
	if !dstVal.CanSet() {
		return newMappingError(path, reflect.ValueOf(raw), dstVal, fmt.Errorf("destination field is not settable"))
	}

	srcVal := reflect.ValueOf(raw)
	if ok, err := ctx.applyConverter(srcVal, dstVal, path); ok {
		return err
	}
	if err := setDefaultScalar(raw, dstVal); err != nil {
		return newMappingError(path, srcVal, dstVal, err)
	}
	return nil
}

func setDefaultScalar(raw string, dstVal reflect.Value) error {
	if dstVal.Kind() == reflect.Pointer {
		if dstVal.IsNil() {
			dstVal.Set(reflect.New(dstVal.Type().Elem()))
		}
		return setDefaultScalar(raw, dstVal.Elem())
	}

	if ok, err := setTextDefault(raw, dstVal); ok {
		return err
	}

	switch dstVal.Kind() {
	case reflect.String:
		dstVal.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return errors.Join(ErrDefaultValue, err)
		}
		dstVal.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, dstVal.Type().Bits())
		if err != nil {
			return errors.Join(ErrDefaultValue, err)
		}
		dstVal.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value, err := strconv.ParseUint(raw, 10, dstVal.Type().Bits())
		if err != nil {
			return errors.Join(ErrDefaultValue, err)
		}
		dstVal.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, dstVal.Type().Bits())
		if err != nil {
			return errors.Join(ErrDefaultValue, err)
		}
		dstVal.SetFloat(value)
	default:
		return fmt.Errorf("%w: cannot assign %q to %s", ErrDefaultValue, raw, dstVal.Type())
	}
	return nil
}

func setTextDefault(raw string, dstVal reflect.Value) (bool, error) {
	if dstVal.Kind() == reflect.Pointer {
		return false, nil
	}

	pointer := reflect.New(dstVal.Type())
	if !pointer.Type().Implements(textUnmarshalerType) {
		return false, nil
	}

	unmarshaler, ok := pointer.Interface().(encoding.TextUnmarshaler)
	if !ok {
		return false, nil
	}
	if err := unmarshaler.UnmarshalText([]byte(raw)); err != nil {
		return true, errors.Join(ErrDefaultValue, err)
	}

	dstVal.Set(pointer.Elem())
	return true, nil
}
