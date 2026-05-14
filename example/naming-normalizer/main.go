package main

import (
	"fmt"
	"strings"

	"github.com/arcgolabs/mapper"
)

func stripLegacyPrefix(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.TrimPrefix(normalized, "u")

	return normalized
}

func main() {
	type LegacySource struct {
		UID   int
		UName string
	}

	type UserDTO struct {
		ID   int
		Name string
	}

	dto, err := mapper.Map[UserDTO](
		LegacySource{UID: 7, UName: "Ada"},
		mapper.WithNameNormalizer(stripLegacyPrefix),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", dto)

	type LegacyMap map[string]any
	payload := LegacyMap{
		"U_Name": "Grace Hopper",
		"u_id":   42,
	}
	mapped, err := mapper.Map[UserDTO](
		payload,
		mapper.WithNameNormalizer(stripLegacyPrefix),
		mapper.WithFallbackTags("json"),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", mapped)
}
