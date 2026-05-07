package mapper

import (
	"errors"
	"fmt"
	"reflect"
)

func (ctx *mappingContext) runBeforeHooks(srcVal, dstVal reflect.Value) error {
	return ctx.runHooks(beforeHook, srcVal, dstVal)
}

func (ctx *mappingContext) runAfterHooks(srcVal, dstVal reflect.Value) error {
	return ctx.runHooks(afterHook, srcVal, dstVal)
}

func (ctx *mappingContext) runHooks(phase hookPhase, srcVal, dstVal reflect.Value) error {
	srcVal = unwrapInterface(srcVal)
	if !srcVal.IsValid() {
		return nil
	}

	hooks := ctx.hooksForPhase(phase).find(srcVal.Type(), dstVal.Type())
	for _, hook := range hooks {
		if err := callHook(hook, srcVal, dstVal); err != nil {
			return fmt.Errorf("mapper: %s hook %s -> %s failed: %w", phaseName(phase), srcVal.Type(), dstVal.Type(), err)
		}
	}
	return nil
}

func (ctx *mappingContext) hooksForPhase(phase hookPhase) hookMap {
	if phase == beforeHook {
		return ctx.beforeHooks()
	}
	return ctx.afterHooks()
}

func (ctx *mappingContext) hasBeforeHooks() bool {
	return ctx.beforeHooks().Len() > 0
}

func (ctx *mappingContext) hasAfterHooks() bool {
	return ctx.afterHooks().Len() > 0
}

func (ctx *mappingContext) beforeHooks() hookMap {
	if ctx.hooks == nil {
		return hookMap{}
	}
	return ctx.hooks.before
}

func (ctx *mappingContext) afterHooks() hookMap {
	if ctx.hooks == nil {
		return hookMap{}
	}
	return ctx.hooks.after
}

func callHook(hook mappingHook, srcVal, dstVal reflect.Value) error {
	out := hook.fn.Call([]reflect.Value{srcVal, dstVal})
	if !hook.hasError || out[0].IsNil() {
		return nil
	}

	err, ok := out[0].Interface().(error)
	if !ok {
		return errors.New("hook returned non-error failure")
	}
	return err
}

func phaseName(phase hookPhase) string {
	if phase == beforeHook {
		return "before-map"
	}
	return "after-map"
}
