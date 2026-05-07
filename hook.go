package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

type hookPhase uint8

const (
	beforeHook hookPhase = iota
	afterHook
)

type hookKey struct {
	src reflect.Type
	dst reflect.Type
}

type mappingHook struct {
	fn       reflect.Value
	hasError bool
}

type hookMap struct {
	items *cxmap.MultiMap[hookKey, mappingHook]
}

func newHookMap(capacity int) hookMap {
	return hookMap{items: cxmap.NewMultiMapWithCapacity[hookKey, mappingHook](capacity)}
}

func (m hookMap) Len() int {
	if m.items == nil {
		return 0
	}
	return m.items.ValueCount()
}

func (m hookMap) Get(key hookKey) ([]mappingHook, bool) {
	if m.items == nil {
		return nil, false
	}
	hooks := m.items.Get(key)
	return hooks, len(hooks) > 0
}

func (m hookMap) Range(fn func(key hookKey, hooks []mappingHook) bool) {
	if m.items == nil {
		return
	}
	m.items.Range(fn)
}

func (m hookMap) Set(key hookKey, hooks []mappingHook) {
	if m.items == nil {
		return
	}
	m.items.Set(key, hooks...)
}

func (m hookMap) Append(key hookKey, hook mappingHook) {
	if m.items == nil {
		return
	}
	m.items.Put(key, hook)
}

type hookSet struct {
	before hookMap
	after  hookMap
}

func newHookSet(before, after hookMap) *hookSet {
	if before.Len() == 0 && after.Len() == 0 {
		return nil
	}
	return &hookSet{before: before, after: after}
}

type hookRegistry struct {
	mu     sync.Mutex
	before atomic.Value
	after  atomic.Value
}

func newHookRegistry() *hookRegistry {
	r := &hookRegistry{}
	r.before.Store(hookMap{})
	r.after.Store(hookMap{})
	return r
}

func (r *hookRegistry) registerBefore(fn any) error {
	return r.register(beforeHook, fn)
}

func (r *hookRegistry) registerAfter(fn any) error {
	return r.register(afterHook, fn)
}

func (r *hookRegistry) register(phase hookPhase, fn any) error {
	key, hook, err := parseHook(fn)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	next := cloneHookMap(r.snapshot(phase))
	next.Append(key, hook)
	r.store(phase, next)
	return nil
}

func parseHook(fn any) (hookKey, mappingHook, error) {
	if fn == nil {
		return hookKey{}, mappingHook{}, errors.New("mapper: hook must not be nil")
	}

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return hookKey{}, mappingHook{}, fmt.Errorf("mapper: hook must be a function, got %s", t)
	}
	if t.NumIn() != 2 {
		return hookKey{}, mappingHook{}, fmt.Errorf("mapper: hook %s must have source and destination inputs", t)
	}
	if t.In(1).Kind() != reflect.Pointer {
		return hookKey{}, mappingHook{}, fmt.Errorf("mapper: hook %s destination input must be a pointer", t)
	}
	if t.NumOut() != 0 && t.NumOut() != 1 {
		return hookKey{}, mappingHook{}, fmt.Errorf("mapper: hook %s must return nothing or error", t)
	}
	if t.NumOut() == 1 && !t.Out(0).Implements(errorType) {
		return hookKey{}, mappingHook{}, fmt.Errorf("mapper: hook %s return value must implement error", t)
	}

	key := hookKey{src: t.In(0), dst: t.In(1)}
	hook := mappingHook{fn: reflect.ValueOf(fn), hasError: t.NumOut() == 1}
	return key, hook, nil
}

func cloneHookMap(source hookMap) hookMap {
	if source.Len() == 0 {
		return newHookMap(1)
	}

	clone := newHookMap(source.Len() + 1)
	source.Range(func(key hookKey, hooks []mappingHook) bool {
		clone.Set(key, append([]mappingHook(nil), hooks...))
		return true
	})
	return clone
}

func (r *hookRegistry) snapshot(phase hookPhase) hookMap {
	if r == nil {
		return hookMap{}
	}
	if phase == beforeHook {
		hooks, ok := r.before.Load().(hookMap)
		if !ok {
			return hookMap{}
		}
		return hooks
	}

	hooks, ok := r.after.Load().(hookMap)
	if !ok {
		return hookMap{}
	}
	return hooks
}

func (r *hookRegistry) store(phase hookPhase, hooks hookMap) {
	if phase == beforeHook {
		r.before.Store(hooks)
		return
	}
	r.after.Store(hooks)
}

func mergeHookMap(base hookMap, fns []any) (hookMap, error) {
	if len(fns) == 0 {
		return base, nil
	}

	merged := cloneHookMap(base)
	for _, fn := range fns {
		key, hook, err := parseHook(fn)
		if err != nil {
			return hookMap{}, err
		}
		merged.Append(key, hook)
	}
	return merged, nil
}

func (m hookMap) find(src, dst reflect.Type) []mappingHook {
	hooks, ok := m.Get(hookKey{src: src, dst: dst})
	if !ok {
		return nil
	}
	return hooks
}
