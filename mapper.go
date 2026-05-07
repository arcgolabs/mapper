package mapper

import (
	"errors"
	"reflect"

	lru "github.com/hashicorp/golang-lru/v2"
)

var defaultMapper = New()

// Mapper maps values between compatible Go types. It caches struct mapping
// plans per source/destination type pair and can hold reusable converters.
type Mapper struct {
	config     Config
	converters *converterRegistry
	hooks      *hookRegistry
	plans      *lru.Cache[planKey, *plan]
}

type mappingContext struct {
	mapper     *Mapper
	config     Config
	converters converterMap
	hooks      *hookSet
}

// New creates a Mapper with optional configuration and reusable converters.
func New(opts ...Option) *Mapper {
	st := newSettings()
	applyOptions(&st, opts)

	m := newMapper(st.config)
	m.mustRegisterSettings(st)
	return m
}

func applyOptions(st *settings, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(st)
		}
	}
}

func newMapper(config Config) *Mapper {
	if config.PlanCacheSize <= 0 {
		config.PlanCacheSize = defaultConfig().PlanCacheSize
	}

	plans, err := lru.New[planKey, *plan](config.PlanCacheSize)
	if err != nil {
		panic(err)
	}

	return &Mapper{
		config:     config,
		converters: newConverterRegistry(),
		hooks:      newHookRegistry(),
		plans:      plans,
	}
}

func (m *Mapper) mustRegisterSettings(st settings) {
	for _, fn := range st.converters {
		if err := m.Register(fn); err != nil {
			panic(err)
		}
	}
	for _, fn := range st.beforeHooks {
		if err := m.RegisterBeforeMap(fn); err != nil {
			panic(err)
		}
	}
	for _, fn := range st.afterHooks {
		if err := m.RegisterAfterMap(fn); err != nil {
			panic(err)
		}
	}
}

// Register adds a reusable converter to this Mapper.
func (m *Mapper) Register(fn any) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}
	return m.converters.register(fn)
}

// RegisterBeforeMap adds a reusable before-map hook to this Mapper.
func (m *Mapper) RegisterBeforeMap(fn any) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}
	return m.hooks.registerBefore(fn)
}

// RegisterAfterMap adds a reusable after-map hook to this Mapper.
func (m *Mapper) RegisterAfterMap(fn any) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}
	return m.hooks.registerAfter(fn)
}

// Register adds a reusable converter to the package-level default Mapper.
func Register[S, D any](fn func(S) D) error {
	return defaultMapper.Register(fn)
}

// RegisterE adds a reusable converter that can return an error to the
// package-level default Mapper.
func RegisterE[S, D any](fn func(S) (D, error)) error {
	return defaultMapper.Register(fn)
}

// MustRegister adds a converter to the package-level default Mapper and panics
// if the converter shape is invalid.
func MustRegister[S, D any](fn func(S) D) {
	if err := Register(fn); err != nil {
		panic(err)
	}
}

// MustRegisterE adds an error-returning converter to the package-level default
// Mapper and panics if the converter shape is invalid.
func MustRegisterE[S, D any](fn func(S) (D, error)) {
	if err := RegisterE(fn); err != nil {
		panic(err)
	}
}

// RegisterBeforeMap adds a reusable before-map hook to the package-level default Mapper.
func RegisterBeforeMap[S, D any](fn func(S, *D)) error {
	return defaultMapper.RegisterBeforeMap(fn)
}

// RegisterBeforeMapE adds an error-returning before-map hook to the default Mapper.
func RegisterBeforeMapE[S, D any](fn func(S, *D) error) error {
	return defaultMapper.RegisterBeforeMap(fn)
}

// RegisterAfterMap adds a reusable after-map hook to the package-level default Mapper.
func RegisterAfterMap[S, D any](fn func(S, *D)) error {
	return defaultMapper.RegisterAfterMap(fn)
}

// RegisterAfterMapE adds an error-returning after-map hook to the default Mapper.
func RegisterAfterMapE[S, D any](fn func(S, *D) error) error {
	return defaultMapper.RegisterAfterMap(fn)
}

// Map maps src into a new destination value.
func Map[D any](src any, opts ...Option) (D, error) {
	var dst D
	err := defaultMapper.MapInto(&dst, src, opts...)
	return dst, err
}

// MustMap maps src into a new destination value and panics on error.
func MustMap[D any](src any, opts ...Option) D {
	dst, err := Map[D](src, opts...)
	if err != nil {
		panic(err)
	}
	return dst
}

// MapInto maps src into dst. Unmatched destination fields are left unchanged
// unless Strict mode is enabled.
func MapInto[D any](dst *D, src any, opts ...Option) error {
	return defaultMapper.MapInto(dst, src, opts...)
}

// Slice maps a source slice or array into a destination slice.
func Slice[D any](src any, opts ...Option) ([]D, error) {
	var dst []D
	err := defaultMapper.MapInto(&dst, src, opts...)
	return dst, err
}

// MapSlice maps a typed source slice into a typed destination slice. D is the
// destination element type, while S is inferred from src.
func MapSlice[D any, S any](src []S, opts ...Option) ([]D, error) {
	return Slice[D](src, opts...)
}

// MapMap maps a typed source map into a destination map while preserving keys.
// D is the destination value type, while K and S are inferred from src.
func MapMap[D any, K comparable, S any](src map[K]S, opts ...Option) (map[K]D, error) {
	return Map[map[K]D](src, opts...)
}

// MapInto maps src into dst. dst must be a non-nil pointer.
func (m *Mapper) MapInto(dst, src any, opts ...Option) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}

	ctx, err := m.newContext(opts)
	if err != nil {
		return err
	}

	dstVal := reflect.ValueOf(dst)
	if !dstVal.IsValid() || dstVal.Kind() != reflect.Pointer || dstVal.IsNil() {
		return errors.New("mapper: destination must be a non-nil pointer")
	}

	srcVal := reflect.ValueOf(src)
	if ctx.hasBeforeHooks() {
		if err := ctx.runBeforeHooks(srcVal, dstVal); err != nil {
			return err
		}
	}
	if err := ctx.mapValue(srcVal, dstVal.Elem(), "$"); err != nil {
		return err
	}
	if ctx.hasAfterHooks() {
		return ctx.runAfterHooks(srcVal, dstVal)
	}
	return nil
}

func (m *Mapper) newContext(opts []Option) (mappingContext, error) {
	beforeHooks := m.hooks.snapshot(beforeHook)
	afterHooks := m.hooks.snapshot(afterHook)
	base := mappingContext{
		mapper:     m,
		config:     m.config,
		converters: m.converters.snapshot(),
		hooks:      newHookSet(beforeHooks, afterHooks),
	}
	if len(opts) == 0 {
		return base, nil
	}

	st := settings{config: m.config}
	applyOptions(&st, opts)

	converters, err := mergeConverterMap(base.converters, st.converters)
	if err != nil {
		return mappingContext{}, err
	}
	beforeHooks, err = mergeHookMap(base.beforeHooks(), st.beforeHooks)
	if err != nil {
		return mappingContext{}, err
	}
	afterHooks, err = mergeHookMap(base.afterHooks(), st.afterHooks)
	if err != nil {
		return mappingContext{}, err
	}

	return mappingContext{
		mapper:     m,
		config:     st.config,
		converters: converters,
		hooks:      newHookSet(beforeHooks, afterHooks),
	}, nil
}

func (m *Mapper) getPlan(srcType, dstType reflect.Type, cfg Config) *plan {
	key := planKey{src: srcType, dst: dstType, tag: cfg.TagName}
	if cached, ok := m.plans.Get(key); ok {
		return cached
	}

	p := buildPlan(srcType, dstType, cfg)
	m.plans.Add(key, p)
	return p
}
