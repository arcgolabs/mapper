package main

import (
	"fmt"
	"strings"

	"github.com/arcgolabs/mapper"
)

func main() {
	type Source struct {
		FullName string
		Role     string
	}

	type UserDTO struct {
		Name  string `mapper:"full-name"`
		Label string `mapper:"role"`
	}

	m := mapper.New(
		mapper.AfterField("name", func(src Source, dst *UserDTO, name *string) {
			*name = strings.ToUpper(*name)
		}),
		mapper.AfterField("label", func(src Source, dst *UserDTO, label *string) {
			*label = "mapped:" + *label
		}),
	)

	var dto UserDTO
	if err := m.MapInto(&dto, Source{
		FullName: "Ada Lovelace",
		Role:     "admin",
	}); err != nil {
		panic(err)
	}

	fmt.Printf("dto: %#v\n", dto)
}
