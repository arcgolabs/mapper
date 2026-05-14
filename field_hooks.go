package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

type fieldHookPhase uint8

const (
	beforeFieldHook fieldHookPhase = iota
	afterFieldHook
)

type fieldHook struct {
	fn       reflect.Value
	hasError bool
}

type fieldHookKey struct {
	src       reflect.Type
	dst       reflect.Type
	fieldName string
	fieldType reflect.Type
}

type fieldHookMap struct {
	items *cxmap.MultiMap[fieldHookKey, fieldHook]
}

func newFieldHookMap(capacity int) fieldHookMap {
	return fieldHookMap{items: cxmap.NewMultiMapWithCapacity[fieldHookKey, fieldHook](capacity)}
}

func (m fieldHookMap) Len() int {
	if m.items == nil {
		return 0
	}
	return m.items.ValueCount()
}

func (m fieldHookMap) Get(key fieldHookKey) ([]fieldHook, bool) {
	if m.items == nil {
		return nil, false
	}
	hooks := m.items.Get(key)
	return hooks, len(hooks) > 0
}

func (m fieldHookMap) Range(fn func(key fieldHookKey, hooks []fieldHook) bool) {
	if m.items == nil {
		return
	}
	m.items.Range(fn)
}

func (m fieldHookMap) Set(key fieldHookKey, hooks []fieldHook) {
	if m.items == nil {
		return
	}
	m.items.Set(key, hooks...)
}

func (m fieldHookMap) Append(key fieldHookKey, hook fieldHook) {
	if m.items == nil {
		return
	}
	m.items.Put(key, hook)
}

func (m fieldHookMap) find(src, dst reflect.Type, fieldName string, fieldType reflect.Type) []fieldHook {
	if m.items == nil {
		return nil
	}
	hooks, ok := m.Get(fieldHookKey{
		src:       src,
		dst:       dst,
		fieldName: normalizeName(fieldName),
		fieldType: fieldType,
	})
	if ok {
		return hooks
	}
	return nil
}

func parseFieldHook(field string, fn any) (fieldHookKey, fieldHook, error) {
	if field == "" {
		return fieldHookKey{}, fieldHook{}, errors.New("mapper: field hook requires a destination field name")
	}
	if fn == nil {
		return fieldHookKey{}, fieldHook{}, errors.New("mapper: field hook must not be nil")
	}

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s must be a function", t)
	}
	if t.NumIn() != 3 {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s must have source, destination-pointer, and destination field-pointer inputs", t)
	}
	if t.In(1).Kind() != reflect.Pointer {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s destination input must be a pointer", t)
	}
	if t.In(2).Kind() != reflect.Pointer {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s field input must be a pointer", t)
	}
	if t.NumOut() != 0 && t.NumOut() != 1 {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s must return nothing or error", t)
	}
	if t.NumOut() == 1 && !t.Out(0).Implements(errorType) {
		return fieldHookKey{}, fieldHook{}, fmt.Errorf("mapper: field hook %s return value must implement error", t)
	}

	key := fieldHookKey{
		src:       t.In(0),
		dst:       t.In(1),
		fieldName: normalizeName(field),
		fieldType: t.In(2),
	}
	hook := fieldHook{fn: reflect.ValueOf(fn), hasError: t.NumOut() == 1}
	return key, hook, nil
}

type fieldHookSet struct {
	before fieldHookMap
	after  fieldHookMap
}

func newFieldHookSet(before, after fieldHookMap) *fieldHookSet {
	if before.Len() == 0 && after.Len() == 0 {
		return nil
	}
	return &fieldHookSet{before: before, after: after}
}

type fieldHookRegistry struct {
	mu     sync.Mutex
	before atomic.Value
	after  atomic.Value
}

func newFieldHookRegistry() *fieldHookRegistry {
	r := &fieldHookRegistry{}
	r.before.Store(fieldHookMap{})
	r.after.Store(fieldHookMap{})
	return r
}

func (r *fieldHookRegistry) registerBefore(field string, fn any) error {
	return r.register(beforeFieldHook, field, fn)
}

func (r *fieldHookRegistry) registerAfter(field string, fn any) error {
	return r.register(afterFieldHook, field, fn)
}

func (r *fieldHookRegistry) register(phase fieldHookPhase, field string, fn any) error {
	key, hook, err := parseFieldHook(field, fn)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	next := cloneFieldHookMap(r.snapshot(phase))
	next.Append(key, hook)
	r.store(phase, next)
	return nil
}

func cloneFieldHookMap(source fieldHookMap) fieldHookMap {
	if source.Len() == 0 {
		return newFieldHookMap(1)
	}

	clone := newFieldHookMap(source.Len() + 1)
	source.Range(func(key fieldHookKey, hooks []fieldHook) bool {
		clone.Set(key, append([]fieldHook(nil), hooks...))
		return true
	})
	return clone
}

func (r *fieldHookRegistry) snapshot(phase fieldHookPhase) fieldHookMap {
	if r == nil {
		return fieldHookMap{}
	}
	if phase == beforeFieldHook {
		hooks, ok := r.before.Load().(fieldHookMap)
		if !ok {
			return fieldHookMap{}
		}
		return hooks
	}

	hooks, ok := r.after.Load().(fieldHookMap)
	if !ok {
		return fieldHookMap{}
	}
	return hooks
}

func (r *fieldHookRegistry) snapshotBefore() fieldHookMap {
	return r.snapshot(beforeFieldHook)
}

func (r *fieldHookRegistry) snapshotAfter() fieldHookMap {
	return r.snapshot(afterFieldHook)
}

func (r *fieldHookRegistry) store(phase fieldHookPhase, hooks fieldHookMap) {
	if phase == beforeFieldHook {
		r.before.Store(hooks)
		return
	}
	r.after.Store(hooks)
}

func mergeFieldHookMap(base fieldHookMap, specs []fieldHookSpec) (fieldHookMap, error) {
	if len(specs) == 0 {
		return base, nil
	}

	merged := cloneFieldHookMap(base)
	for _, spec := range specs {
		key, hook, err := parseFieldHook(spec.field, spec.fn)
		if err != nil {
			return fieldHookMap{}, err
		}
		merged.Append(key, hook)
	}
	return merged, nil
}

func (m *fieldHookSet) beforeHooks() fieldHookMap {
	if m == nil {
		return fieldHookMap{}
	}
	return m.before
}

func (m *fieldHookSet) afterHooks() fieldHookMap {
	if m == nil {
		return fieldHookMap{}
	}
	return m.after
}

func (m *fieldHookSet) find(phase fieldHookPhase, src, dst reflect.Type, field string, fieldType reflect.Type) []fieldHook {
	if m == nil {
		return nil
	}
	hooks := m.hooksForPhase(phase)
	return hooks.find(src, dst, field, fieldType)
}

func (m *fieldHookSet) hooksForPhase(phase fieldHookPhase) fieldHookMap {
	if m == nil {
		return fieldHookMap{}
	}
	if phase == beforeFieldHook {
		return m.before
	}
	return m.after
}

func (m *fieldHookSet) hasHooks() bool {
	if m == nil {
		return false
	}
	return m.before.Len() > 0 || m.after.Len() > 0
}

func (m fieldHookMap) findCompatible(src, dst reflect.Type, fieldName string, fieldType reflect.Type) []fieldHook {
	hooks := m.find(src, dst, fieldName, fieldType)
	if len(hooks) > 0 {
		return hooks
	}

	if src == nil {
		return nil
	}

	normalized := normalizeName(fieldName)
	compatible := make([]fieldHook, 0)
	m.Range(func(key fieldHookKey, value []fieldHook) bool {
		if key.dst != dst {
			return true
		}
		if key.fieldName != normalized || key.fieldType != fieldType {
			return true
		}
		if src.AssignableTo(key.src) {
			compatible = append(compatible, value...)
		}
		return true
	})
	if len(compatible) == 0 {
		return nil
	}
	return compatible
}

func (m *fieldHookSet) findForSourceAndField(phase fieldHookPhase, src, dst reflect.Type, field string, fieldType reflect.Type) []fieldHook {
	hooks := m.hooksForPhase(phase)
	if m == nil || src == nil {
		return nil
	}

	fieldName := normalizeName(field)
	h := hooks.findCompatible(src, dst, fieldName, fieldType)
	if len(h) > 0 {
		return h
	}

	for src.Kind() == reflect.Pointer {
		src = src.Elem()
		h = hooks.findCompatible(src, dst, fieldName, fieldType)
		if len(h) > 0 {
			return h
		}
	}

	return nil
}
