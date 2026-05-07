package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	cxmap "github.com/arcgolabs/collectionx/mapping"
)

var errorType = reflect.TypeFor[error]()

type conversionKey struct {
	src reflect.Type
	dst reflect.Type
}

type converter struct {
	fn       reflect.Value
	hasError bool
}

type converterMap struct {
	items *cxmap.Map[conversionKey, converter]
}

func newConverterMap(capacity int) converterMap {
	return converterMap{items: cxmap.NewMapWithCapacity[conversionKey, converter](capacity)}
}

func (m converterMap) Len() int {
	if m.items == nil {
		return 0
	}
	return m.items.Len()
}

func (m converterMap) Get(key conversionKey) (converter, bool) {
	if m.items == nil {
		return converter{}, false
	}
	return m.items.Get(key)
}

func (m converterMap) Set(key conversionKey, conv converter) {
	if m.items == nil {
		return
	}
	m.items.Set(key, conv)
}

func (m converterMap) Range(fn func(key conversionKey, conv converter) bool) {
	if m.items == nil {
		return
	}
	m.items.Range(fn)
}

type converterRegistry struct {
	mu         sync.Mutex
	converters atomic.Value
}

func newConverterRegistry() *converterRegistry {
	r := &converterRegistry{}
	r.converters.Store(converterMap{})
	return r
}

func (r *converterRegistry) register(fn any) error {
	key, conv, err := parseConverter(fn)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	next := cloneConverterMap(r.snapshot())
	next.Set(key, conv)
	r.converters.Store(next)
	return nil
}

func parseConverter(fn any) (conversionKey, converter, error) {
	if fn == nil {
		return conversionKey{}, converter{}, errors.New("mapper: converter must not be nil")
	}

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return conversionKey{}, converter{}, fmt.Errorf("mapper: converter must be a function, got %s", t)
	}
	if t.NumIn() != 1 {
		return conversionKey{}, converter{}, fmt.Errorf("mapper: converter %s must have exactly one input", t)
	}
	if t.NumOut() != 1 && t.NumOut() != 2 {
		return conversionKey{}, converter{}, fmt.Errorf("mapper: converter %s must return one value or value plus error", t)
	}
	if t.NumOut() == 2 && !t.Out(1).Implements(errorType) {
		return conversionKey{}, converter{}, fmt.Errorf("mapper: converter %s second return value must implement error", t)
	}

	key := conversionKey{src: t.In(0), dst: t.Out(0)}
	conv := converter{fn: reflect.ValueOf(fn), hasError: t.NumOut() == 2}
	return key, conv, nil
}

func cloneConverterMap(source converterMap) converterMap {
	if source.Len() == 0 {
		return newConverterMap(1)
	}

	clone := newConverterMap(source.Len() + 1)
	source.Range(func(key conversionKey, conv converter) bool {
		clone.Set(key, conv)
		return true
	})
	return clone
}

func mergeConverterMap(base converterMap, fns []any) (converterMap, error) {
	if len(fns) == 0 {
		return base, nil
	}

	merged := cloneConverterMap(base)
	for _, fn := range fns {
		key, conv, err := parseConverter(fn)
		if err != nil {
			return converterMap{}, err
		}
		merged.Set(key, conv)
	}
	return merged, nil
}

func (r *converterRegistry) snapshot() converterMap {
	if r == nil {
		return converterMap{}
	}

	converters, ok := r.converters.Load().(converterMap)
	if !ok {
		return converterMap{}
	}
	return converters
}

func (m converterMap) find(src, dst reflect.Type) (converter, bool) {
	if m.Len() == 0 {
		return converter{}, false
	}
	return m.Get(conversionKey{src: src, dst: dst})
}

func (m converterMap) findKey(key conversionKey) (converter, bool) {
	if m.Len() == 0 {
		return converter{}, false
	}
	return m.Get(key)
}
