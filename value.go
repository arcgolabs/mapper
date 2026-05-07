package mapper

import (
	"fmt"
	"reflect"
)

func (ctx *mappingContext) mapValue(srcVal, dstVal reflect.Value, path string) error {
	if !dstVal.CanSet() {
		return fmt.Errorf("mapper: %s is not settable", path)
	}

	srcVal = unwrapInterface(srcVal)
	if ctx.mapZero(srcVal, dstVal) {
		return nil
	}
	if ok, err := ctx.applyConverter(srcVal, dstVal, path); ok {
		return err
	}
	if assignValue(srcVal, dstVal) {
		return nil
	}
	if ok, err := ctx.mapPointer(srcVal, dstVal, path); ok {
		return err
	}

	return ctx.mapComposite(srcVal, dstVal, path)
}

func (ctx *mappingContext) mapZero(srcVal, dstVal reflect.Value) bool {
	if !srcVal.IsValid() || isNil(srcVal) {
		dstVal.SetZero()
		return true
	}
	return false
}

func assignValue(srcVal, dstVal reflect.Value) bool {
	if srcVal.Type().AssignableTo(dstVal.Type()) {
		dstVal.Set(srcVal)
		return true
	}
	if canConvert(srcVal.Type(), dstVal.Type()) {
		dstVal.Set(srcVal.Convert(dstVal.Type()))
		return true
	}
	return false
}

func (ctx *mappingContext) mapPointer(srcVal, dstVal reflect.Value, path string) (bool, error) {
	if dstVal.Kind() == reflect.Pointer {
		if dstVal.IsNil() {
			dstVal.Set(reflect.New(dstVal.Type().Elem()))
		}
		return true, ctx.mapValue(srcVal, dstVal.Elem(), path)
	}

	if srcVal.Kind() == reflect.Pointer {
		if srcVal.IsNil() {
			dstVal.SetZero()
			return true, nil
		}
		return true, ctx.mapValue(srcVal.Elem(), dstVal, path)
	}

	return false, nil
}

func (ctx *mappingContext) mapComposite(srcVal, dstVal reflect.Value, path string) error {
	dstKind := dstVal.Kind()
	srcKind := srcVal.Kind()

	if dstKind == reflect.Slice && isSliceLike(srcKind) {
		return ctx.mapSlice(srcVal, dstVal, path)
	}
	if dstKind == reflect.Map && srcKind == reflect.Map {
		return ctx.mapMap(srcVal, dstVal, path)
	}
	if dstKind == reflect.Struct && srcKind == reflect.Struct {
		return ctx.mapStruct(srcVal, dstVal, path)
	}

	return fmt.Errorf("mapper: %s cannot map %s to %s", path, srcVal.Type(), dstVal.Type())
}

func isSliceLike(kind reflect.Kind) bool {
	return kind == reflect.Slice || kind == reflect.Array
}

func (ctx *mappingContext) applyConverter(srcVal, dstVal reflect.Value, path string) (bool, error) {
	if ctx.converters.Len() == 0 {
		return false, nil
	}

	conv, ok := ctx.findConverter(srcVal.Type(), dstVal.Type())
	if !ok {
		return false, nil
	}

	return true, ctx.applyKnownConverter(conv, srcVal, dstVal, path)
}

func (ctx *mappingContext) applyKnownConverter(conv converter, srcVal, dstVal reflect.Value, path string) error {
	out := conv.fn.Call([]reflect.Value{srcVal})
	if conv.hasError && !out[1].IsNil() {
		err, ok := out[1].Interface().(error)
		if !ok {
			return fmt.Errorf("mapper: %s converter returned non-error failure", path)
		}
		return fmt.Errorf("mapper: %s converter %s -> %s failed: %w", path, srcVal.Type(), dstVal.Type(), err)
	}

	value := out[0]
	if assignValue(value, dstVal) {
		return nil
	}

	return fmt.Errorf("mapper: %s converter returned %s, cannot assign to %s", path, value.Type(), dstVal.Type())
}

func (ctx *mappingContext) findConverter(src, dst reflect.Type) (converter, bool) {
	return ctx.converters.find(src, dst)
}

func (ctx *mappingContext) mapStruct(srcVal, dstVal reflect.Value, path string) error {
	p := ctx.mapper.getPlan(srcVal.Type(), dstVal.Type(), ctx.config)
	if ctx.config.Strict && len(p.missing) > 0 {
		return &MissingFieldsError{
			Source:      srcVal.Type(),
			Destination: dstVal.Type(),
			Fields:      append([]string(nil), p.missing...),
		}
	}

	for _, step := range p.steps {
		if err := ctx.mapField(step, srcVal, dstVal, path); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *mappingContext) mapField(step fieldStep, srcVal, dstVal reflect.Value, path string) error {
	srcField, ok := valueByIndex(srcVal, step.srcIndex)
	if !ok {
		return nil
	}

	dstField, ok := settableFieldByIndex(dstVal, step.dstIndex)
	if !ok {
		return fmt.Errorf("mapper: %s.%s is not settable", path, step.dstName)
	}

	fieldPath := path + "." + step.dstName
	if ctx.converters.Len() == 0 {
		return ctx.mapPlannedValueWithoutConverter(step.op, srcField, dstField, fieldPath)
	}
	return ctx.mapPlannedFieldValue(step, srcField, dstField, fieldPath)
}

func (ctx *mappingContext) mapPlannedValue(op valueOp, srcVal, dstVal reflect.Value, path string) error {
	srcVal = unwrapInterface(srcVal)
	if ctx.mapZero(srcVal, dstVal) {
		return nil
	}
	if ok, err := ctx.applyConverter(srcVal, dstVal, path); ok {
		return err
	}

	return ctx.mapWithoutConverter(op, srcVal, dstVal, path)
}

func (ctx *mappingContext) mapPlannedFieldValue(step fieldStep, srcVal, dstVal reflect.Value, path string) error {
	if step.op == opDynamic {
		return ctx.mapPlannedValue(step.op, srcVal, dstVal, path)
	}

	srcVal = unwrapInterface(srcVal)
	if ctx.mapZero(srcVal, dstVal) {
		return nil
	}
	if conv, ok := ctx.converters.findKey(step.convKey); ok {
		return ctx.applyKnownConverter(conv, srcVal, dstVal, path)
	}
	return ctx.mapWithoutConverter(step.op, srcVal, dstVal, path)
}

func (ctx *mappingContext) mapPlannedValueWithoutConverter(op valueOp, srcVal, dstVal reflect.Value, path string) error {
	srcVal = unwrapInterface(srcVal)
	if ctx.mapZero(srcVal, dstVal) {
		return nil
	}
	return ctx.mapWithoutConverter(op, srcVal, dstVal, path)
}

func (ctx *mappingContext) mapWithoutConverter(op valueOp, srcVal, dstVal reflect.Value, path string) error {
	switch op {
	case opAssign:
		dstVal.Set(srcVal)
		return nil
	case opConvert:
		dstVal.Set(srcVal.Convert(dstVal.Type()))
		return nil
	case opPointer:
		_, err := ctx.mapPointer(srcVal, dstVal, path)
		return err
	case opSlice:
		return ctx.mapSlice(srcVal, dstVal, path)
	case opMap:
		return ctx.mapMap(srcVal, dstVal, path)
	case opStruct:
		return ctx.mapStruct(srcVal, dstVal, path)
	case opDynamic:
		return ctx.mapValue(srcVal, dstVal, path)
	default:
		return ctx.mapValue(srcVal, dstVal, path)
	}
}
