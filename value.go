package mapper

import (
	"errors"
	"fmt"
	"reflect"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

type fieldExecution struct {
	step         fieldStep
	converter    converter
	hasConverter bool
}

type executionPlanMap struct {
	items *cxmap.MultiMap[*plan, fieldExecution]
}

func newExecutionPlanMap() executionPlanMap {
	return executionPlanMap{items: cxmap.NewMultiMap[*plan, fieldExecution]()}
}

func (m executionPlanMap) Get(p *plan) ([]fieldExecution, bool) {
	if m.items == nil {
		return nil, false
	}
	steps := m.items.Get(p)
	return steps, len(steps) > 0
}

func (m executionPlanMap) Set(p *plan, steps []fieldExecution) {
	if m.items == nil {
		return
	}
	m.items.Set(p, steps...)
}

func (ctx *mappingContext) mapValue(srcVal, dstVal reflect.Value, path string) error {
	if !dstVal.CanSet() {
		return newMappingError(path, srcVal, dstVal, fmt.Errorf("%s is not settable", path))
	}

	srcVal = unwrapInterface(srcVal)
	if ctx.shouldIgnoreSource(srcVal) {
		return nil
	}
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

func (ctx *mappingContext) shouldIgnoreSource(srcVal reflect.Value) bool {
	if ctx == nil {
		return false
	}
	if ctx.config.IgnoreNil && (!srcVal.IsValid() || isNil(srcVal)) {
		return true
	}
	if !ctx.config.IgnoreZero {
		return false
	}
	if !srcVal.IsValid() || isNil(srcVal) {
		return true
	}
	return srcVal.IsZero()
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
	if dstKind == reflect.Struct && srcKind == reflect.Map && srcVal.Type().Key().Kind() == reflect.String {
		return ctx.mapStringMapToStruct(srcVal, dstVal, path)
	}
	if dstKind == reflect.Struct && srcKind == reflect.Struct {
		return ctx.mapStruct(srcVal, dstVal, path)
	}

	return newMappingError(path, srcVal, dstVal, ErrCannotMap)
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
			return newMappingError(path, srcVal, dstVal, fmt.Errorf("converter returned non-error failure"))
		}
		return newMappingError(path, srcVal, dstVal, err)
	}

	value := out[0]
	if assignValue(value, dstVal) {
		return nil
	}

	return newMappingError(path, srcVal, dstVal, fmt.Errorf("converter returned %s, cannot assign to %s", value.Type(), dstVal.Type()))
}

func (ctx *mappingContext) findConverter(src, dst reflect.Type) (converter, bool) {
	return ctx.converters.find(src, dst)
}

func (ctx *mappingContext) mapStruct(srcVal, dstVal reflect.Value, path string) error {
	p := ctx.mapper.getPlan(srcVal.Type(), dstVal.Type(), ctx.config)
	if len(p.requiredMissing) > 0 {
		return &MissingFieldsError{
			Source:      srcVal.Type(),
			Destination: dstVal.Type(),
			Fields:      append([]string(nil), p.requiredMissing...),
			Required:    true,
		}
	}
	if ctx.config.Strict && len(p.missing) > 0 {
		return &MissingFieldsError{
			Source:      srcVal.Type(),
			Destination: dstVal.Type(),
			Fields:      append([]string(nil), p.missing...),
		}
	}

	if ctx.converters.Len() != 0 {
		if path != "$" {
			for _, exec := range ctx.executionPlan(p) {
				if err := ctx.mapFieldExecution(exec, srcVal, dstVal, path); err != nil {
					return err
				}
			}
			return nil
		}

		for _, step := range p.steps {
			if err := ctx.mapFieldWithConverterLookup(step, srcVal, dstVal, path); err != nil {
				return err
			}
		}
		return nil
	}

	for _, step := range p.steps {
		if err := ctx.mapField(step, srcVal, dstVal, path); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *mappingContext) mapField(step fieldStep, srcVal, dstVal reflect.Value, path string) error {
	srcField, dstField, ok, err := ctx.resolveFieldValues(step, srcVal, dstVal, path)
	if err != nil || !ok {
		return err
	}

	fieldPath := path + "." + step.dstName
	srcField = unwrapInterface(srcField)
	if ctx.shouldUseDefault(step, srcField) {
		return ctx.mapDefaultValue(step.defaultValue, dstField, fieldPath)
	}
	return ctx.mapPlannedValueWithoutConverter(step.op, srcField, dstField, fieldPath)
}

func (ctx *mappingContext) mapFieldWithConverterLookup(step fieldStep, srcVal, dstVal reflect.Value, path string) error {
	srcField, dstField, ok, err := ctx.resolveFieldValues(step, srcVal, dstVal, path)
	if err != nil || !ok {
		return err
	}

	return ctx.mapPlannedFieldValue(step, srcField, dstField, path+"."+step.dstName)
}

func (ctx *mappingContext) mapFieldExecution(exec fieldExecution, srcVal, dstVal reflect.Value, path string) error {
	step := exec.step
	srcField, dstField, ok, err := ctx.resolveFieldValues(step, srcVal, dstVal, path)
	if err != nil || !ok {
		return err
	}

	fieldPath := path + "." + step.dstName
	if exec.hasConverter {
		srcField = unwrapInterface(srcField)
		if ctx.shouldUseDefault(step, srcField) {
			return ctx.mapDefaultValue(step.defaultValue, dstField, fieldPath)
		}
		if ctx.shouldIgnoreSource(srcField) {
			return nil
		}
		if ctx.mapZero(srcField, dstField) {
			return nil
		}
		return ctx.applyKnownConverter(exec.converter, srcField, dstField, fieldPath)
	}
	if step.op == opDynamic {
		return ctx.mapPlannedValue(step.op, srcField, dstField, fieldPath)
	}
	return ctx.mapPlannedValueWithoutConverter(step.op, srcField, dstField, fieldPath)
}

func (ctx *mappingContext) resolveFieldValues(step fieldStep, srcVal, dstVal reflect.Value, path string) (reflect.Value, reflect.Value, bool, error) {
	dstField, ok := settableFieldByIndex(dstVal, step.dstIndex)
	if !ok {
		fieldPath := path + "." + step.dstName
		return reflect.Value{}, reflect.Value{}, false, newMappingError(fieldPath, reflect.Value{}, dstVal, fmt.Errorf("destination field is not settable"))
	}

	if !step.hasSource {
		if step.hasDefault {
			fieldPath := path + "." + step.dstName
			return reflect.Value{}, dstField, false, ctx.mapDefaultValue(step.defaultValue, dstField, fieldPath)
		}
		if step.required {
			return reflect.Value{}, dstField, false, &MissingFieldsError{
				Source:      srcVal.Type(),
				Destination: dstVal.Type(),
				Fields:      []string{step.dstName},
				Required:    true,
			}
		}
		return reflect.Value{}, dstField, false, nil
	}

	srcField, ok := valueByIndex(srcVal, step.srcIndex)
	if !ok {
		if step.hasDefault {
			fieldPath := path + "." + step.dstName
			return reflect.Value{}, dstField, false, ctx.mapDefaultValue(step.defaultValue, dstField, fieldPath)
		}
		if step.required {
			return reflect.Value{}, dstField, false, &MissingFieldsError{
				Source:      srcVal.Type(),
				Destination: dstVal.Type(),
				Fields:      []string{step.dstName},
				Required:    true,
			}
		}
		return reflect.Value{}, dstField, false, nil
	}

	return srcField, dstField, true, nil
}

func (ctx *mappingContext) mapPlannedValue(op valueOp, srcVal, dstVal reflect.Value, path string) error {
	srcVal = unwrapInterface(srcVal)
	if ctx.shouldIgnoreSource(srcVal) {
		return nil
	}
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
	if ctx.shouldUseDefault(step, srcVal) {
		return ctx.mapDefaultValue(step.defaultValue, dstVal, path)
	}
	if ctx.shouldIgnoreSource(srcVal) {
		return nil
	}
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
	if ctx.shouldIgnoreSource(srcVal) {
		return nil
	}
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

func (ctx *mappingContext) executionPlan(p *plan) []fieldExecution {
	if ctx.executionPlans.items == nil {
		ctx.executionPlans = newExecutionPlanMap()
	}
	if cached, ok := ctx.executionPlans.Get(p); ok {
		return cached
	}

	steps := make([]fieldExecution, 0, len(p.steps))
	for _, step := range p.steps {
		exec := fieldExecution{step: step}
		if step.op != opDynamic {
			if conv, ok := ctx.converters.findKey(step.convKey); ok {
				exec.converter = conv
				exec.hasConverter = true
			}
		}
		steps = append(steps, exec)
	}
	ctx.executionPlans.Set(p, steps)
	return steps
}

func (ctx *mappingContext) shouldUseDefault(step fieldStep, srcVal reflect.Value) bool {
	if !step.hasDefault {
		return false
	}
	if !srcVal.IsValid() || isNil(srcVal) {
		return true
	}
	return srcVal.IsZero()
}

func newMappingError(path string, srcVal, dstVal reflect.Value, cause error) error {
	var srcType reflect.Type
	if srcVal.IsValid() {
		srcType = srcVal.Type()
	}
	var dstType reflect.Type
	if dstVal.IsValid() {
		dstType = dstVal.Type()
	}
	var existing *MappingError
	if cause != nil && errors.As(cause, &existing) {
		return cause
	}
	return &MappingError{
		Path:            path,
		SourceType:      srcType,
		DestinationType: dstType,
		Cause:           cause,
	}
}
