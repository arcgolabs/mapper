package mapper

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"

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
	keyIndex := indexStringMapKeys(srcVal, ctx.config)
	usedKeys := cxmap.NewMap[string, bool]()
	steps, missing, required := ctx.buildMapStructSteps(srcVal, dstVal.Type(), keyIndex, usedKeys)
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
	if ctx.config.StrictDynamicMapKeys {
		if unknown := findUnusedMapKeys(keyIndex, usedKeys); len(unknown) > 0 {
			return &UnknownFieldsError{
				Source:      srcVal.Type(),
				Destination: dstVal.Type(),
				Fields:      unknown,
			}
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
		dstHookVal, hasDstHook := mappingFieldContainer(dstVal)
		if hasDstHook {
			if err := ctx.runFieldHooks(beforeFieldHook, srcVal, dstHookVal, dstField.Addr(), fieldPath, step.field.Name); err != nil {
				return err
			}
		}

		mapped := false
		if step.spec.hasDefault && (!srcField.IsValid() || isNil(srcField) || srcField.IsZero()) {
			if err := ctx.mapDefaultValue(step.spec.defaultValue, dstField, fieldPath); err != nil {
				return err
			}
			mapped = true
		} else if ctx.shouldIgnoreSource(srcField) {
			atomic.AddUint64(&ctx.mapper.metrics.ConditionChecks, 1)
			atomic.AddUint64(&ctx.mapper.metrics.ConditionSkips, 1)
		} else if ctx.mapZero(srcField, dstField) {
			mapped = true
		} else {
			op := opDynamic
			if srcField.IsValid() {
				op = compileValueOp(srcField.Type(), step.field.Type)
			}
			if ctx.converters.Len() == 0 {
				if err := ctx.mapPlannedValueWithoutConverter(op, srcField, dstField, fieldPath); err != nil {
					return err
				}
			} else {
				if err := ctx.mapPlannedValue(op, srcField, dstField, fieldPath); err != nil {
					return err
				}
			}
			mapped = true
		}

		if !mapped || !hasDstHook {
			continue
		}
		if err := ctx.runFieldHooks(afterFieldHook, srcVal, dstHookVal, dstField.Addr(), fieldPath, step.field.Name); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *mappingContext) buildMapStructSteps(
	srcVal reflect.Value,
	dstType reflect.Type,
	keyIndex *cxmap.Map[string, reflect.Value],
	usedTopLevelKeys *cxmap.Map[string, bool],
) ([]mapStructStep, []string, []string) {
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
		if ok {
			ctx.recordUsedTopLevelMapKey(spec.sourceName, usedTopLevelKeys, ctx.config)
		}
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

func (ctx *mappingContext) recordUsedTopLevelMapKey(name string, usedTopLevelKeys *cxmap.Map[string, bool], cfg Config) {
	top, ok := topLevelMapKey(cfg, name)
	if !ok || usedTopLevelKeys == nil {
		return
	}
	usedTopLevelKeys.Set(top, true)
}

func topLevelMapKey(cfg Config, path string) (string, bool) {
	if path == "" {
		return "", false
	}
	part := strings.SplitN(path, ".", 2)[0]
	part = strings.TrimSpace(part)
	if part == "" {
		return "", false
	}
	return normalizeWithConfig(cfg, part), true
}

func findUnusedMapKeys(keys *cxmap.Map[string, reflect.Value], used *cxmap.Map[string, bool]) []string {
	if keys == nil {
		return nil
	}

	var missing []string
	keys.Range(func(key string, value reflect.Value) bool {
		if used != nil {
			if _, ok := used.Get(key); ok {
				return true
			}
		}
		missing = append(missing, value.String())
		return true
	})
	sort.Strings(missing)
	return missing
}

func indexStringMapKeys(value reflect.Value, cfg Config) *cxmap.Map[string, reflect.Value] {
	out := cxmap.NewMap[string, reflect.Value]()
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key()
		normalized := normalizeWithConfig(cfg, key.String())
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
	if key, ok := keyIndex.Get(normalizeWithConfig(cfg, name)); ok {
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
			keyIndex := indexStringMapKeys(current, cfg)
			key, ok := keyIndex.Get(normalizeWithConfig(cfg, part))
			if !ok {
				return reflect.Value{}, false
			}
			current = current.MapIndex(key)
		case reflect.Struct:
			fields := collectSourceFields(current.Type(), cfg)
			field, ok := fields.Get(normalizeWithConfig(cfg, part))
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
