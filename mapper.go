package mapper

import (
	"errors"
	"reflect"
	"strings"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
)

var defaultMapper = New()

// Mapper maps values between compatible Go types. It caches struct mapping
// plans per source/destination type pair and can hold reusable converters.
type Mapper struct {
	config     Config
	converters *converterRegistry
	hooks      *hookRegistry
	fieldHooks *fieldHookRegistry
	metrics    MappingMetrics
	plans      *lru.Cache[planKey, *plan]
	validator  ValidationEngine
}

type mappingContext struct {
	mapper         *Mapper
	config         Config
	converters     converterMap
	hooks          *hookSet
	fieldHooks     *fieldHookSet
	validator      ValidationEngine
	executionPlans executionPlanMap
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
		fieldHooks: newFieldHookRegistry(),
		plans:      plans,
	}
}

func (m *Mapper) mustRegisterSettings(st settings) {
	m.validator = st.validator
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
	for _, spec := range st.beforeFieldHooks {
		if err := m.RegisterBeforeField(spec.field, spec.fn); err != nil {
			panic(err)
		}
	}
	for _, spec := range st.afterFieldHooks {
		if err := m.RegisterAfterField(spec.field, spec.fn); err != nil {
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

// RegisterBeforeField adds a reusable before field hook to this Mapper.
func (m *Mapper) RegisterBeforeField(field string, fn any) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}
	return m.fieldHooks.registerBefore(field, fn)
}

// RegisterAfterField adds a reusable after field hook to this Mapper.
func (m *Mapper) RegisterAfterField(field string, fn any) error {
	if m == nil {
		return errors.New("mapper: nil Mapper")
	}
	return m.fieldHooks.registerAfter(field, fn)
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

// RegisterBeforeField adds a reusable before field hook to the package-level default Mapper.
func RegisterBeforeField[S, D, F any](field string, fn func(S, *D, *F)) error {
	return defaultMapper.RegisterBeforeField(field, fn)
}

// RegisterBeforeFieldE adds an error-returning before field hook to the package-level
// default Mapper.
func RegisterBeforeFieldE[S, D, F any](field string, fn func(S, *D, *F) error) error {
	return defaultMapper.RegisterBeforeField(field, fn)
}

// RegisterAfterField adds a reusable after field hook to the package-level default Mapper.
func RegisterAfterField[S, D, F any](field string, fn func(S, *D, *F)) error {
	return defaultMapper.RegisterAfterField(field, fn)
}

// RegisterAfterFieldE adds an error-returning after field hook to the package-level
// default Mapper.
func RegisterAfterFieldE[S, D, F any](field string, fn func(S, *D, *F) error) error {
	return defaultMapper.RegisterAfterField(field, fn)
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
	atomic.AddUint64(&m.metrics.MappingCalls, 1)

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
		if err := ctx.runAfterHooks(srcVal, dstVal); err != nil {
			return err
		}
	}

	if err := ctx.validate(dstVal); err != nil {
		return err
	}

	return nil
}

func (m *Mapper) newContext(opts []Option) (mappingContext, error) {
	beforeHooks := m.hooks.snapshot(beforeHook)
	afterHooks := m.hooks.snapshot(afterHook)
	beforeFieldHooks := m.fieldHooks.snapshot(beforeFieldHook)
	afterFieldHooks := m.fieldHooks.snapshot(afterFieldHook)
	base := mappingContext{
		mapper:     m,
		config:     m.config,
		converters: m.converters.snapshot(),
		hooks:      newHookSet(beforeHooks, afterHooks),
		fieldHooks: newFieldHookSet(beforeFieldHooks, afterFieldHooks),
		validator:  m.validator,
	}
	if len(opts) == 0 {
		return base, nil
	}

	st := settings{
		config:    m.config,
		validator: m.validator,
	}
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
	beforeFieldHooks, err = mergeFieldHookMap(base.fieldHooks.beforeHooks(), st.beforeFieldHooks)
	if err != nil {
		return mappingContext{}, err
	}
	afterFieldHooks, err = mergeFieldHookMap(base.fieldHooks.afterHooks(), st.afterFieldHooks)
	if err != nil {
		return mappingContext{}, err
	}

	return mappingContext{
		mapper:     m,
		config:     st.config,
		converters: converters,
		hooks:      newHookSet(beforeHooks, afterHooks),
		fieldHooks: newFieldHookSet(beforeFieldHooks, afterFieldHooks),
		validator:  st.validator,
	}, nil
}

func (m *Mapper) getPlan(srcType, dstType reflect.Type, cfg Config) *plan {
	key := planKey{
		src:          srcType,
		dst:          dstType,
		tag:          tagCacheKey(cfg),
		normalizerID: cfg.nameNormalizerID,
	}
	if cached, ok := m.plans.Get(key); ok {
		atomic.AddUint64(&m.metrics.PlanCacheHits, 1)
		return cached
	}
	atomic.AddUint64(&m.metrics.PlanCacheMisses, 1)

	p := buildPlan(srcType, dstType, cfg)
	atomic.AddUint64(&m.metrics.PlanCacheStores, 1)
	m.plans.Add(key, p)
	return p
}

// MappingMetrics summarizes runtime cache and execution counters.
type MappingMetrics struct {
	PlanCacheHits       uint64
	PlanCacheMisses     uint64
	PlanCacheStores     uint64
	MappingCalls        uint64
	ExecutionPlanHits   uint64
	ExecutionPlanMisses uint64
	ConditionChecks     uint64
	ConditionSkips      uint64
	FieldHookRuns       uint64
}

// Metrics returns a snapshot of metrics for this Mapper.
func (m *Mapper) Metrics() MappingMetrics {
	return MappingMetrics{
		PlanCacheHits:       atomic.LoadUint64(&m.metrics.PlanCacheHits),
		PlanCacheMisses:     atomic.LoadUint64(&m.metrics.PlanCacheMisses),
		PlanCacheStores:     atomic.LoadUint64(&m.metrics.PlanCacheStores),
		MappingCalls:        atomic.LoadUint64(&m.metrics.MappingCalls),
		ExecutionPlanHits:   atomic.LoadUint64(&m.metrics.ExecutionPlanHits),
		ExecutionPlanMisses: atomic.LoadUint64(&m.metrics.ExecutionPlanMisses),
		ConditionChecks:     atomic.LoadUint64(&m.metrics.ConditionChecks),
		ConditionSkips:      atomic.LoadUint64(&m.metrics.ConditionSkips),
		FieldHookRuns:       atomic.LoadUint64(&m.metrics.FieldHookRuns),
	}
}

// ResetMetrics resets runtime counters for this Mapper.
func (m *Mapper) ResetMetrics() {
	atomic.StoreUint64(&m.metrics.PlanCacheHits, 0)
	atomic.StoreUint64(&m.metrics.PlanCacheMisses, 0)
	atomic.StoreUint64(&m.metrics.PlanCacheStores, 0)
	atomic.StoreUint64(&m.metrics.MappingCalls, 0)
	atomic.StoreUint64(&m.metrics.ExecutionPlanHits, 0)
	atomic.StoreUint64(&m.metrics.ExecutionPlanMisses, 0)
	atomic.StoreUint64(&m.metrics.ConditionChecks, 0)
	atomic.StoreUint64(&m.metrics.ConditionSkips, 0)
	atomic.StoreUint64(&m.metrics.FieldHookRuns, 0)
}

func tagCacheKey(cfg Config) string {
	normalizerID := cfg.nameNormalizerID
	if normalizerID == "" {
		normalizerID = "default"
	}
	if len(cfg.FallbackTagNames) == 0 {
		return cfg.TagName + "\x00" + normalizerID
	}
	return cfg.TagName + "\x00" + normalizerID + "\x00" + strings.Join(cfg.FallbackTagNames, "\x00")
}
