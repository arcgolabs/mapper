package main

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type UserDTO struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	input := map[string]any{
		"id":   7,
		"name": "Ada",
		"role": "admin",
	}

	loose, err := mapper.Map[UserDTO](input, mapper.WithFallbackTags("json"))
	fmt.Printf("loose: %#v err=%v\n", loose, err)

	_, err = mapper.Map[UserDTO](
		input,
		mapper.WithFallbackTags("json"),
		mapper.WithStrictDynamicMapKeys(true),
	)
	var unknown *mapper.UnknownFieldsError
	if errors.As(err, &unknown) {
		fmt.Printf("strict unknown keys: %v\n", unknown.Fields)
		return
	}

	fmt.Printf("strict: err=%v\n", err)
}
