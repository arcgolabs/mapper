package mapper

import "reflect"

type indexRange int

func (r indexRange) Len() int {
	if r < 0 {
		return 0
	}
	return int(r)
}

func (r indexRange) Range(fn func(index int, item int) bool) {
	if fn == nil {
		return
	}
	for i := range int(r) {
		if !fn(i, i) {
			return
		}
	}
}

type mapEntry struct {
	key   reflect.Value
	value reflect.Value
}

type reflectMapEntries struct {
	value reflect.Value
}

func (e reflectMapEntries) Len() int {
	if !e.value.IsValid() {
		return 0
	}
	return e.value.Len()
}

func (e reflectMapEntries) Range(fn func(index int, item mapEntry) bool) {
	if fn == nil || !e.value.IsValid() {
		return
	}

	iter := e.value.MapRange()
	index := 0
	for iter.Next() {
		if !fn(index, mapEntry{key: iter.Key(), value: iter.Value()}) {
			return
		}
		index++
	}
}
