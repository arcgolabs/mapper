package mapper

import (
	"reflect"
	"strings"
	"unicode"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

type planKey struct {
	src reflect.Type
	dst reflect.Type
	tag string
}

type plan struct {
	steps           []fieldStep
	missing         []string
	requiredMissing []string
}

type fieldStep struct {
	srcIndex     []int
	dstIndex     []int
	srcName      string
	dstName      string
	convKey      conversionKey
	op           valueOp
	hasSource    bool
	required     bool
	defaultValue string
	hasDefault   bool
}

type sourceField struct {
	index []int
	typ   reflect.Type
}

type valueOp uint8

const (
	opDynamic valueOp = iota
	opAssign
	opConvert
	opPointer
	opSlice
	opMap
	opStruct
)

func buildPlan(srcType, dstType reflect.Type, cfg Config) *plan {
	sourceFields := collectSourceFields(srcType, cfg)
	fields := reflect.VisibleFields(dstType)

	p := &plan{}
	for _, field := range fields {
		if !isExported(field) {
			continue
		}

		spec := destinationFieldSpec(field, cfg)
		if spec.skip {
			continue
		}

		src, ok := resolveSourceField(srcType, sourceFields, spec.sourceName, cfg)
		if !ok {
			if spec.hasDefault {
				p.steps = append(p.steps, fieldStep{
					dstIndex:     field.Index,
					srcName:      spec.sourceName,
					dstName:      field.Name,
					required:     spec.required,
					defaultValue: spec.defaultValue,
					hasDefault:   true,
				})
				continue
			}
			if spec.required {
				p.requiredMissing = append(p.requiredMissing, field.Name)
			}
			p.missing = append(p.missing, field.Name)
			continue
		}

		p.steps = append(p.steps, fieldStep{
			srcIndex:     src.index,
			dstIndex:     field.Index,
			srcName:      spec.sourceName,
			dstName:      field.Name,
			convKey:      conversionKey{src: src.typ, dst: field.Type},
			op:           compileValueOp(src.typ, field.Type),
			hasSource:    true,
			required:     spec.required,
			defaultValue: spec.defaultValue,
			hasDefault:   spec.hasDefault,
		})
	}

	return p
}

func compileValueOp(srcType, dstType reflect.Type) valueOp {
	if isDynamicPlanType(srcType, dstType) {
		return opDynamic
	}
	if srcType.AssignableTo(dstType) {
		return opAssign
	}
	if canConvert(srcType, dstType) {
		return opConvert
	}
	return compileCompositeOp(srcType.Kind(), dstType.Kind())
}

func isDynamicPlanType(srcType, dstType reflect.Type) bool {
	return srcType.Kind() == reflect.Interface || dstType.Kind() == reflect.Interface
}

func compileCompositeOp(srcKind, dstKind reflect.Kind) valueOp {
	if srcKind == reflect.Pointer || dstKind == reflect.Pointer {
		return opPointer
	}
	if dstKind == reflect.Slice && isSliceLike(srcKind) {
		return opSlice
	}
	if dstKind == reflect.Map && srcKind == reflect.Map {
		return opMap
	}
	if dstKind == reflect.Struct && srcKind == reflect.Struct {
		return opStruct
	}
	return opDynamic
}

func collectSourceFields(t reflect.Type, cfg Config) *cxmap.Map[string, sourceField] {
	t = derefType(t)
	out := cxmap.NewMap[string, sourceField]()
	if t.Kind() != reflect.Struct {
		return out
	}

	for _, field := range reflect.VisibleFields(t) {
		if !isExported(field) {
			continue
		}

		spec := sourceFieldSpec(field, cfg)
		if spec.skip {
			continue
		}

		info := sourceField{index: field.Index, typ: field.Type}
		addSourceAlias(out, field.Name, info)
		for _, alias := range spec.aliases {
			addSourceAlias(out, alias, info)
		}
	}

	return out
}

func addSourceAlias(fields *cxmap.Map[string, sourceField], name string, info sourceField) {
	key := normalizeName(name)
	if key == "" {
		return
	}
	if _, exists := fields.Get(key); !exists {
		fields.Set(key, info)
	}
}

func resolveSourceField(srcType reflect.Type, sourceFields *cxmap.Map[string, sourceField], name string, cfg Config) (sourceField, bool) {
	if strings.Contains(name, ".") {
		return resolveSourcePath(srcType, name, cfg)
	}

	return sourceFields.Get(normalizeName(name))
}

func resolveSourcePath(srcType reflect.Type, path string, cfg Config) (sourceField, bool) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return sourceField{}, false
	}

	var index []int
	current := derefType(srcType)
	var typ reflect.Type
	for _, part := range parts {
		fields := collectSourceFields(current, cfg)
		field, ok := fields.Get(normalizeName(part))
		if !ok {
			return sourceField{}, false
		}

		index = append(index, field.index...)
		typ = field.typ
		current = derefType(field.typ)
	}

	return sourceField{index: index, typ: typ}, true
}

type tagSpec struct {
	name         string
	sourceName   string
	aliases      []string
	skip         bool
	hasName      bool
	hasTag       bool
	required     bool
	defaultValue string
	hasDefault   bool
}

func sourceFieldSpec(field reflect.StructField, cfg Config) tagSpec {
	spec := parseTag(field.Tag.Get(cfg.TagName))
	if spec.skip {
		return spec
	}

	if spec.hasName {
		spec.aliases = append(spec.aliases, spec.name)
	}
	for _, tagName := range cfg.FallbackTagNames {
		if name, ok := parseTagName(field.Tag.Get(tagName)); ok {
			spec.aliases = append(spec.aliases, name)
		}
	}
	return spec
}

func destinationFieldSpec(field reflect.StructField, cfg Config) tagSpec {
	spec := parseTag(field.Tag.Get(cfg.TagName))
	if spec.skip {
		return spec
	}

	spec.sourceName = field.Name
	if spec.hasName {
		spec.sourceName = spec.name
		return spec
	}

	for _, tagName := range cfg.FallbackTagNames {
		if name, ok := parseTagName(field.Tag.Get(tagName)); ok {
			spec.sourceName = name
			return spec
		}
	}
	return spec
}

func parseTag(tag string) tagSpec {
	if tag == "" {
		return tagSpec{}
	}

	spec := tagSpec{hasTag: true}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "-" {
		spec.skip = true
		return spec
	}
	if name != "" {
		spec.name = name
		spec.hasName = true
	}

	for _, option := range parts[1:] {
		option = strings.TrimSpace(option)
		switch {
		case option == "required":
			spec.required = true
		case strings.HasPrefix(option, "default="):
			spec.defaultValue = strings.TrimPrefix(option, "default=")
			spec.hasDefault = true
		}
	}

	return spec
}

func parseTagName(tag string) (string, bool) {
	if tag == "" {
		return "", false
	}

	name := strings.TrimSpace(strings.Split(tag, ",")[0])
	if name == "" || name == "-" {
		return "", false
	}
	return name, true
}

func isExported(field reflect.StructField) bool {
	return field.PkgPath == ""
}

func normalizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r == ' ' || r == '.' {
			return -1
		}
		return unicode.ToLower(r)
	}, name)
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
