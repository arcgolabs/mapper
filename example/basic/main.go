package main

import (
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type Source struct {
		ID   int
		Name string
	}

	type UserDTO struct {
		ID   int
		Name string
	}

	dto, err := mapper.Map[UserDTO](Source{ID: 1, Name: "Ada"})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", dto)
}
