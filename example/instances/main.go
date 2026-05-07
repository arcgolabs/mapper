package main

import (
	"fmt"

	"github.com/arcgolabs/mapper"
)

type Source struct {
	ID int
}

type DTO struct {
	ID int
}

func main() {
	// Package-level API with per-call options.
	packageLevel, err := mapper.Map[DTO](Source{ID: 1}, mapper.Converter(func(v int) int {
		return v + 1
	}))
	if err != nil {
		panic(err)
	}
	fmt.Println("package-level:", packageLevel.ID)

	// Reusable mapper instance with registration.
	reusable := mapper.New()
	if err := reusable.Register(func(v int) int {
		return v * 10
	}); err != nil {
		panic(err)
	}
	if err := reusable.RegisterBeforeMap(func(src Source, dst *DTO) {
		dst.ID = 1
	}); err != nil {
		panic(err)
	}
	if err := reusable.RegisterAfterMap(func(src Source, dst *DTO) {
		_ = src
		dst.ID += 0
	}); err != nil {
		panic(err)
	}

	var reusableDst DTO
	if err := reusable.MapInto(&reusableDst, Source{ID: 2}); err != nil {
		panic(err)
	}
	fmt.Println("reusable:", reusableDst.ID)
}
