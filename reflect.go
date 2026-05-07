package mapper

import (
	"reflect"
)

func unwrapInterface(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func isNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	kind := v.Kind()
	if kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface ||
		kind == reflect.Map || kind == reflect.Pointer || kind == reflect.Slice {
		return v.IsNil()
	}
	return false
}

func valueByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for _, i := range index {
		v = unwrapInterface(v)
		if !v.IsValid() {
			return reflect.Value{}, false
		}
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		v = v.Field(i)
	}
	return v, true
}

func settableFieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for pos, i := range index {
		parent, ok := derefForSet(v)
		if !ok || parent.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}

		v = parent.Field(i)
		if pos == len(index)-1 {
			return v, v.CanSet()
		}
	}

	return v, v.CanSet()
}

func derefForSet(v reflect.Value) (reflect.Value, bool) {
	v = unwrapInterface(v)
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if !v.CanSet() {
				return reflect.Value{}, false
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	return v, true
}

func canConvert(src, dst reflect.Type) bool {
	if !src.ConvertibleTo(dst) {
		return false
	}

	srcKind := src.Kind()
	dstKind := dst.Kind()
	if srcKind == dstKind && isScalar(srcKind) {
		return true
	}

	if isNumeric(srcKind) && isNumeric(dstKind) {
		return true
	}

	return false
}

func isScalar(kind reflect.Kind) bool {
	return kind == reflect.Bool || kind == reflect.String
}

func isNumeric(kind reflect.Kind) bool {
	return (kind >= reflect.Int && kind <= reflect.Int64) ||
		(kind >= reflect.Uint && kind <= reflect.Uintptr) ||
		(kind >= reflect.Float32 && kind <= reflect.Complex128)
}
