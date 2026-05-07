package mapper_test

import (
	"testing"
	"time"

	"github.com/arcgolabs/mapper"
)

type benchUser struct {
	ID        int
	Name      string
	Email     string
	CreatedAt time.Time
}

type benchUserDTO struct {
	ID        int
	Name      string
	Email     string
	CreatedAt string
}

func BenchmarkMapWarmPlan(b *testing.B) {
	src := benchUser{
		ID:        1,
		Name:      "Ada",
		Email:     "ada@example.com",
		CreatedAt: time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC),
	}
	m := mapper.New(mapper.Converter(func(v time.Time) string {
		return v.Format(time.RFC3339)
	}))

	var warm benchUserDTO
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst benchUserDTO
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMapWarmPlanNoConverter(b *testing.B) {
	type source struct {
		ID    int
		Name  string
		Email string
		Rank  int64
	}
	type destination struct {
		ID    int
		Name  string
		Email string
		Rank  int64
	}

	src := source{ID: 1, Name: "Ada", Email: "ada@example.com", Rank: 10}
	m := mapper.New()

	var warm destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSliceWarmPlan(b *testing.B) {
	src := []benchUser{
		{ID: 1, Name: "Ada", Email: "ada@example.com", CreatedAt: time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)},
		{ID: 2, Name: "Grace", Email: "grace@example.com", CreatedAt: time.Date(2026, 5, 7, 10, 31, 0, 0, time.UTC)},
		{ID: 3, Name: "Lin", Email: "lin@example.com", CreatedAt: time.Date(2026, 5, 7, 10, 32, 0, 0, time.UTC)},
	}
	m := mapper.New(mapper.Converter(func(v time.Time) string {
		return v.Format(time.RFC3339)
	}))

	var warm []benchUserDTO
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst []benchUserDTO
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNestedStructWarmPlan(b *testing.B) {
	type profile struct {
		Name  string
		Email string
	}
	type source struct {
		ID      int
		Profile profile
	}
	type destination struct {
		ID      int
		Profile profile
	}

	src := source{ID: 1, Profile: profile{Name: "Ada", Email: "ada@example.com"}}
	m := mapper.New()

	var warm destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeSliceWarmPlanNoConverter(b *testing.B) {
	type source struct {
		ID    int
		Name  string
		Email string
	}
	type destination struct {
		ID    int
		Name  string
		Email string
	}

	src := make([]source, 128)
	for i := range src {
		src[i] = source{ID: i, Name: "Ada", Email: "ada@example.com"}
	}
	m := mapper.New()

	var warm []destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst []destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMapStringAnyToStruct(b *testing.B) {
	type destination struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `mapper:"profile.email"`
	}

	src := map[string]any{
		"id":   1,
		"name": "Ada",
		"profile": map[string]any{
			"email": "ada@example.com",
		},
	}
	m := mapper.New(mapper.WithFallbackTags("json"))

	var warm destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConverterHeavyWarmPlan(b *testing.B) {
	type source struct {
		A int
		B int
		C int
		D int
	}
	type destination struct {
		A string
		B string
		C string
		D string
	}

	src := source{A: 1, B: 2, C: 3, D: 4}
	m := mapper.New(mapper.Converter(func(v int) string {
		return time.Unix(int64(v), 0).UTC().Format("15:04:05")
	}))

	var warm destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidationWarmPlan(b *testing.B) {
	type source struct {
		ID   int
		Name string
	}
	type destination struct {
		ID   int
		Name string
	}

	src := source{ID: 1, Name: "Ada"}
	m := mapper.New(mapper.WithValidator(mapper.ValidationFunc(func(any) error {
		return nil
	})))

	var warm destination
	if err := m.MapInto(&warm, src); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var dst destination
		if err := m.MapInto(&dst, src); err != nil {
			b.Fatal(err)
		}
	}
}
