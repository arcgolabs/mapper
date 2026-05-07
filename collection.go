package mapper

import (
	"fmt"
	"reflect"

	cxlist "github.com/arcgolabs/collectionx/list"
)

func (ctx *mappingContext) mapSlice(srcVal, dstVal reflect.Value, path string) error {
	if srcVal.Kind() == reflect.Slice && srcVal.IsNil() {
		dstVal.SetZero()
		return nil
	}

	out := reflect.MakeSlice(dstVal.Type(), srcVal.Len(), srcVal.Len())
	indices := indexRange(srcVal.Len())
	mapped, err := cxlist.ReduceErrList(indices, out, func(acc reflect.Value, _ int, item int) (reflect.Value, error) {
		if err := ctx.mapValue(srcVal.Index(item), acc.Index(item), fmt.Sprintf("%s[%d]", path, item)); err != nil {
			return acc, err
		}
		return acc, nil
	})
	if err != nil {
		return fmt.Errorf("mapper: %s slice mapping failed: %w", path, err)
	}

	dstVal.Set(mapped)
	return nil
}

func (ctx *mappingContext) mapMap(srcVal, dstVal reflect.Value, path string) error {
	if srcVal.IsNil() {
		dstVal.SetZero()
		return nil
	}

	out := reflect.MakeMapWithSize(dstVal.Type(), srcVal.Len())
	entries := reflectMapEntries{value: srcVal}
	mapped, err := cxlist.ReduceErrList(entries, out, func(acc reflect.Value, _ int, entry mapEntry) (reflect.Value, error) {
		dstKey := reflect.New(dstVal.Type().Key()).Elem()
		if err := ctx.mapValue(entry.key, dstKey, path+"[key]"); err != nil {
			return acc, err
		}

		dstValue := reflect.New(dstVal.Type().Elem()).Elem()
		if err := ctx.mapValue(entry.value, dstValue, path+"[value]"); err != nil {
			return acc, err
		}

		acc.SetMapIndex(dstKey, dstValue)
		return acc, nil
	})
	if err != nil {
		return fmt.Errorf("mapper: %s map mapping failed: %w", path, err)
	}

	dstVal.Set(mapped)
	return nil
}
