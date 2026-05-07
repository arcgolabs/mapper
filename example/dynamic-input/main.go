package main

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type UserDTO struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `mapper:"profile.email"`
		Role  string `mapper:",default=user"`
	}

	dto, err := mapper.Map[UserDTO](
		map[string]any{
			"id":   float64(7),
			"name": "Ada",
			"profile": map[string]any{
				"email": "ada@example.com",
			},
		},
		mapper.WithFallbackTags("json"),
	)
	fmt.Printf("dynamic: %+v err=%v\n", dto, err)

	_, err = mapper.Map[UserDTO](
		map[string]any{
			"id": []string{"not", "a", "number"},
		},
		mapper.WithFallbackTags("json"),
	)

	var mappingErr *mapper.MappingError
	if errors.As(err, &mappingErr) {
		fmt.Printf("mapping error: path=%s src=%s dst=%s\n", mappingErr.Path, mappingErr.SourceType, mappingErr.DestinationType)
	}
}
