package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
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

func (ctx *mappingContext) runFieldHooks(
	phase fieldHookPhase,
	srcVal,
	dstVal,
	fieldVal reflect.Value,
	fieldPath string,
	field string,
) error {
	if ctx.fieldHooks == nil {
		return nil
	}

	hooks := ctx.fieldHooksForPhase(phase).findForSourceAndField(
		phase,
		srcVal.Type(),
		dstVal.Type(),
		field,
		fieldVal.Type(),
	)
	if len(hooks) == 0 {
		return nil
	}
	for _, hook := range hooks {
		if err := callFieldHook(hook, srcVal, dstVal, fieldVal); err != nil {
			return newMappingErrorWithKind(
				fieldPath,
				srcVal,
				dstVal,
				MappingErrorKindHook,
				fmt.Errorf("%s hook %s -> %s.%s failed: %w", fieldHookPhaseName(phase), srcVal.Type(), dstVal.Type(), field, err),
			)
		}
		atomic.AddUint64(&ctx.mapper.metrics.FieldHookRuns, 1)
	}
	return nil
}

func (ctx *mappingContext) fieldHooksForPhase(phase fieldHookPhase) *fieldHookSet {
	_ = phase
	return ctx.fieldHooks
}

func (ctx *mappingContext) hasFieldHooks() bool {
	if ctx.fieldHooks == nil {
		return false
	}
	return ctx.fieldHooks.hasHooks()
}

func callFieldHook(hook fieldHook, srcVal, dstVal, fieldVal reflect.Value) error {
	out := hook.fn.Call([]reflect.Value{srcVal, dstVal, fieldVal})
	if !hook.hasError || out[0].IsNil() {
		return nil
	}

	err, ok := out[0].Interface().(error)
	if !ok {
		return errors.New("field hook returned non-error failure")
	}
	return err
}

func phaseName(phase hookPhase) string {
	if phase == beforeHook {
		return "before-map"
	}
	return "after-map"
}

func fieldHookPhaseName(phase fieldHookPhase) string {
	if phase == beforeFieldHook {
		return "before-field"
	}
	return "after-field"
}
