package mapper

import (
	"reflect"
	"testing"
)

func TestNewContextMergesLocalConverters(t *testing.T) {
	m := New()
	ctx, err := m.newContext([]Option{
		Converter(func(v string) string {
			return "converted:" + v
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := ctx.converters.find(reflect.TypeFor[string](), reflect.TypeFor[string]())
	if !ok {
		t.Fatal("expected local string converter in context snapshot")
	}
}

func TestMapPlannedValueUsesLocalConverter(t *testing.T) {
	m := New()
	ctx, err := m.newContext([]Option{
		Converter(func(v string) string {
			return "converted:" + v
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	src := reflect.ValueOf("Ada")
	dst := reflect.New(reflect.TypeFor[string]()).Elem()
	if err := ctx.mapPlannedValue(opAssign, src, dst, "$.Name"); err != nil {
		t.Fatal(err)
	}
	if dst.String() != "converted:Ada" {
		t.Fatalf("unexpected value: %s", dst.String())
	}
}
