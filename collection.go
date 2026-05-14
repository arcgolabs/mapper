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

	if ctx.config.UpdateStrategy == UpdateMerge {
		out := reflect.MakeSlice(dstVal.Type(), 0, dstVal.Len()+srcVal.Len())
		out = reflect.AppendSlice(out, dstVal)
		indices := indexRange(srcVal.Len())
		op := compileValueOp(srcVal.Type().Elem(), dstVal.Type().Elem())
		mapped, err := cxlist.ReduceErrList(indices, out, func(acc reflect.Value, _ int, item int) (reflect.Value, error) {
			dstItem := reflect.New(dstVal.Type().Elem()).Elem()
			itemPath := fmt.Sprintf("%s[%d]", path, item)
			var err error
			if ctx.converters.Len() == 0 {
				err = ctx.mapPlannedValueWithoutConverter(op, srcVal.Index(item), dstItem, itemPath)
			} else {
				err = ctx.mapPlannedValue(op, srcVal.Index(item), dstItem, itemPath)
			}
			if err != nil {
				return acc, err
			}
			acc = reflect.Append(acc, dstItem)
			return acc, nil
		})
		if err != nil {
			return fmt.Errorf("mapper: %s slice mapping failed: %w", path, err)
		}
		dstVal.Set(mapped)
		return nil
	}

	out := reflect.MakeSlice(dstVal.Type(), srcVal.Len(), srcVal.Len())
	indices := indexRange(srcVal.Len())
	op := compileValueOp(srcVal.Type().Elem(), dstVal.Type().Elem())
	mapped, err := cxlist.ReduceErrList(indices, out, func(acc reflect.Value, _ int, item int) (reflect.Value, error) {
		itemPath := fmt.Sprintf("%s[%d]", path, item)
		var err error
		if ctx.converters.Len() == 0 {
			err = ctx.mapPlannedValueWithoutConverter(op, srcVal.Index(item), acc.Index(item), itemPath)
		} else {
			err = ctx.mapPlannedValue(op, srcVal.Index(item), acc.Index(item), itemPath)
		}
		if err != nil {
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
	if ctx.config.UpdateStrategy == UpdateMerge {
		if dstVal.IsNil() {
			dstVal.Set(reflect.MakeMapWithSize(dstVal.Type(), srcVal.Len()))
		}

		entries := reflectMapEntries{value: srcVal}
		keyOp := compileValueOp(srcVal.Type().Key(), dstVal.Type().Key())
		elemOp := compileValueOp(srcVal.Type().Elem(), dstVal.Type().Elem())
		mapped, err := cxlist.ReduceErrList(entries, dstVal, func(acc reflect.Value, _ int, entry mapEntry) (reflect.Value, error) {
			dstKey := reflect.New(dstVal.Type().Key()).Elem()
			var err error
			if ctx.converters.Len() == 0 {
				err = ctx.mapPlannedValueWithoutConverter(keyOp, entry.key, dstKey, path+"[key]")
			} else {
				err = ctx.mapPlannedValue(keyOp, entry.key, dstKey, path+"[key]")
			}
			if err != nil {
				return acc, err
			}

			dstValue := reflect.New(dstVal.Type().Elem()).Elem()
			if ctx.converters.Len() == 0 {
				err = ctx.mapPlannedValueWithoutConverter(elemOp, entry.value, dstValue, path+"[value]")
			} else {
				err = ctx.mapPlannedValue(elemOp, entry.value, dstValue, path+"[value]")
			}
			if err != nil {
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

	out := reflect.MakeMapWithSize(dstVal.Type(), srcVal.Len())
	entries := reflectMapEntries{value: srcVal}
	keyOp := compileValueOp(srcVal.Type().Key(), dstVal.Type().Key())
	elemOp := compileValueOp(srcVal.Type().Elem(), dstVal.Type().Elem())
	mapped, err := cxlist.ReduceErrList(entries, out, func(acc reflect.Value, _ int, entry mapEntry) (reflect.Value, error) {
		dstKey := reflect.New(dstVal.Type().Key()).Elem()
		var err error
		if ctx.converters.Len() == 0 {
			err = ctx.mapPlannedValueWithoutConverter(keyOp, entry.key, dstKey, path+"[key]")
		} else {
			err = ctx.mapPlannedValue(keyOp, entry.key, dstKey, path+"[key]")
		}
		if err != nil {
			return acc, err
		}

		dstValue := reflect.New(dstVal.Type().Elem()).Elem()
		if ctx.converters.Len() == 0 {
			err = ctx.mapPlannedValueWithoutConverter(elemOp, entry.value, dstValue, path+"[value]")
		} else {
			err = ctx.mapPlannedValue(elemOp, entry.value, dstValue, path+"[value]")
		}
		if err != nil {
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
