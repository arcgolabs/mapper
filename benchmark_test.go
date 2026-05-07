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
