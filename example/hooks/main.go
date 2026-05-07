package main

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type Source struct {
		FirstName string
		LastName  string
		ID        int
	}
	type UserDTO struct {
		FirstName string
		LastName  string
		FullName  string
		Label     string
		ID        int
	}

	fmt.Println("local before/after hooks:")
	m := mapper.New(
		mapper.BeforeMap(func(src Source, dst *UserDTO) {
			dst.ID = src.ID * 10
		}),
		mapper.AfterMap(func(src Source, dst *UserDTO) {
			dst.FullName = src.FirstName + " " + src.LastName
			dst.Label = "mapped"
		}),
	)

	var dto UserDTO
	if err := m.MapInto(&dto, Source{FirstName: "Ada", LastName: "Lovelace", ID: 1}); err != nil {
		panic(err)
	}
	fmt.Println(dto.ID, dto.FullName, dto.Label)

	fmt.Println("registered after-map hook:")
	m = mapper.New()
	if err := m.RegisterAfterMap(func(src Source, dst *UserDTO) {
		dst.Label = fmt.Sprintf("U-%d", src.ID)
	}); err != nil {
		panic(err)
	}

	var dto2 UserDTO
	if err := m.MapInto(&dto2, Source{ID: 3}); err != nil {
		panic(err)
	}
	fmt.Println(dto2.Label)

	fmt.Println("error before-map hook:")
	m = mapper.New(
		mapper.BeforeMapE(func(src Source, dst *UserDTO) error {
			if src.ID == 0 {
				return errors.New("id is required")
			}
			return nil
		}),
	)

	err := m.MapInto(&dto2, Source{ID: 0})
	fmt.Println(err)
}
