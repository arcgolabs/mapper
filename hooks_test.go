package mapper_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/arcgolabs/mapper"
)

func TestAfterMapHook(t *testing.T) {
	type user struct {
		FirstName string
		LastName  string
	}
	type userDTO struct {
		FirstName string
		LastName  string
		FullName  string
	}

	m := mapper.New(mapper.AfterMap(func(src user, dst *userDTO) {
		dst.FullName = src.FirstName + " " + src.LastName
	}))

	var dto userDTO
	if err := m.MapInto(&dto, user{FirstName: "Ada", LastName: "Lovelace"}); err != nil {
		t.Fatal(err)
	}

	if dto.FullName != "Ada Lovelace" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestBeforeMapHookError(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID int
	}

	sentinel := errors.New("blocked")
	m := mapper.New(mapper.BeforeMapE(func(src user, dst *userDTO) error {
		if src.ID == 0 {
			return sentinel
		}
		dst.ID = 10
		return nil
	}))

	var dto userDTO
	err := m.MapInto(&dto, user{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected hook error, got %v", err)
	}
}

func TestMapperRegisterAfterMap(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID    int
		Label string
	}

	m := mapper.New()
	if err := m.RegisterAfterMap(func(src user, dst *userDTO) {
		dst.Label = fmt.Sprintf("U-%d", src.ID)
	}); err != nil {
		t.Fatal(err)
	}

	var dto userDTO
	if err := m.MapInto(&dto, user{ID: 42}); err != nil {
		t.Fatal(err)
	}

	if dto.Label != "U-42" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}
