package mapper

import (
	"fmt"
	"reflect"
	"strings"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

type mapStructStep struct {
	field      reflect.StructField
	spec       tagSpec
	source     reflect.Value
	hasSource  bool
	missing    bool
	isRequired bool
}

func (ctx *mappingContext) mapStringMapToStruct(srcVal, dstVal reflect.Value, path string) error {
	steps, missing, required := ctx.buildMapStructSteps(srcVal, dstVal.Type())
	if len(required) > 0 {
		return &MissingFieldsError{
			Source:      srcVal.Type(),
			Destination: dstVal.Type(),
			Fields:      required,
			Required:    true,
		}
	}
	if ctx.config.Strict && len(missing) > 0 {
		return &MissingFieldsError{
			Source:      srcVal.Type(),
			Destination: dstVal.Type(),
			Fields:      missing,
		}
	}

	for _, step := range steps {
		fieldPath := path + "." + step.field.Name
		dstField, ok := settableFieldByIndex(dstVal, step.field.Index)
		if !ok {
			return newMappingError(fieldPath, reflect.Value{}, dstVal, fmt.Errorf("destination field is not settable"))
		}

		if !step.hasSource {
			if step.spec.hasDefault {
				if err := ctx.mapDefaultValue(step.spec.defaultValue, dstField, fieldPath); err != nil {
					return err
				}
			}
			continue
		}

		srcField := unwrapInterface(step.source)
		if step.spec.hasDefault && (!srcField.IsValid() || isNil(srcField) || srcField.IsZero()) {
			if err := ctx.mapDefaultValue(step.spec.defaultValue, dstField, fieldPath); err != nil {
				return err
			}
			continue
		}

		op := opDynamic
		if srcField.IsValid() {
			op = compileValueOp(srcField.Type(), step.field.Type)
		}
		if ctx.converters.Len() == 0 {
			if err := ctx.mapPlannedValueWithoutConverter(op, srcField, dstField, fieldPath); err != nil {
				return err
			}
			continue
		}
		if err := ctx.mapPlannedValue(op, srcField, dstField, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *mappingContext) buildMapStructSteps(srcVal reflect.Value, dstType reflect.Type) ([]mapStructStep, []string, []string) {
	keyIndex := indexStringMapKeys(srcVal)
	fields := reflect.VisibleFields(dstType)
	steps := make([]mapStructStep, 0, len(fields))
	var missing []string
	var required []string

	for _, field := range fields {
		if !isExported(field) {
			continue
		}

		spec := destinationFieldSpec(field, ctx.config)
		if spec.skip {
			continue
		}

		source, ok := lookupNamedMapValue(srcVal, keyIndex, spec.sourceName, ctx.config)
		step := mapStructStep{field: field, spec: spec, source: source, hasSource: ok}
		steps = append(steps, step)
		if ok || spec.hasDefault {
			continue
		}
		if spec.required {
			required = append(required, field.Name)
		}
		missing = append(missing, field.Name)
	}

	return steps, missing, required
}

func indexStringMapKeys(value reflect.Value) *cxmap.Map[string, reflect.Value] {
	out := cxmap.NewMap[string, reflect.Value]()
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key()
		normalized := normalizeName(key.String())
		if normalized == "" {
			continue
		}
		if _, exists := out.Get(normalized); !exists {
			out.Set(normalized, key)
		}
	}
	return out
}

func lookupNamedMapValue(srcVal reflect.Value, keyIndex *cxmap.Map[string, reflect.Value], name string, cfg Config) (reflect.Value, bool) {
	if key, ok := keyIndex.Get(normalizeName(name)); ok {
		return srcVal.MapIndex(key), true
	}
	if !strings.Contains(name, ".") {
		return reflect.Value{}, false
	}
	return lookupMapPath(srcVal, strings.Split(name, "."), cfg)
}

func lookupMapPath(current reflect.Value, parts []string, cfg Config) (reflect.Value, bool) {
	for _, part := range parts {
		current = unwrapInterface(current)
		if !current.IsValid() || isNil(current) {
			return reflect.Value{}, false
		}

		for current.Kind() == reflect.Pointer {
			if current.IsNil() {
				return reflect.Value{}, false
			}
			current = current.Elem()
		}

		switch current.Kind() {
		case reflect.Map:
			if current.Type().Key().Kind() != reflect.String {
				return reflect.Value{}, false
			}
			keyIndex := indexStringMapKeys(current)
			key, ok := keyIndex.Get(normalizeName(part))
			if !ok {
				return reflect.Value{}, false
			}
			current = current.MapIndex(key)
		case reflect.Struct:
			fields := collectSourceFields(current.Type(), cfg)
			field, ok := fields.Get(normalizeName(part))
			if !ok {
				return reflect.Value{}, false
			}
			value, ok := valueByIndex(current, field.index)
			if !ok {
				return reflect.Value{}, false
			}
			current = value
		default:
			return reflect.Value{}, false
		}
	}
	return current, true
}
