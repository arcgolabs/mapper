package main

import (
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type User struct {
		Name  string
		Email string
		Age   int
		Role  string
	}

	type UserPatch struct {
		Name  *string
		Email *string
		Age   int
	}

	name := "Grace"
	user := User{Name: "Ada", Email: "ada@example.com", Age: 36, Role: "admin"}
	err := mapper.MapInto(&user, UserPatch{Name: &name}, mapper.IgnoreNil(), mapper.IgnoreZero())
	fmt.Printf("patch: %+v err=%v\n", user, err)

	type CreateInput struct {
		Name string
	}
	type CreateDTO struct {
		Name   string `mapper:",required"`
		Role   string `mapper:",default=user"`
		Active bool   `mapper:",default=true"`
	}

	dto, err := mapper.Map[CreateDTO](CreateInput{Name: "Lin"})
	fmt.Printf("create: %+v err=%v\n", dto, err)
}
