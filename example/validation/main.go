package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arcgolabs/mapper"
	"github.com/go-playground/validator/v10"
)

func main() {
	type Source struct {
		Name  string
		Email string
		Age   int
	}

	type DTO struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,gte=18"`
	}

	type TaggedSource struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	type TaggedDTO struct {
		Name  string `json:"name" validate:"required,min=3"`
		Email string `json:"email" validate:"required,mycorp"`
		Age   int    `json:"age" validate:"required,gte=18"`
	}

	fmt.Println("1) built-in validation by option:")
	v := validator.New()
	_, err := mapper.Map[DTO](Source{Name: "Al", Email: "bad-email", Age: 16}, mapper.WithValidator(v))
	fmt.Printf("err=%v\n", err)

	fmt.Println("2) mapped value with built-in validation:")
	dto, err := mapper.Map[DTO](Source{Name: "Ada", Email: "ada@example.com", Age: 22}, mapper.WithValidator(v))
	fmt.Printf("mapped=%#v err=%v\n", dto, err)

	fmt.Println("3) register custom validator once and share it on a mapper instance:")
	myCorp := validator.New()
	_ = myCorp.RegisterValidation("mycorp", func(fl validator.FieldLevel) bool {
		return strings.HasSuffix(strings.ToLower(fl.Field().String()), "@mycorp.example")
	})
	mapperWithCustom := mapper.New(
		mapper.WithTagName("json"),
		mapper.WithValidator(myCorp),
	)

	var dst TaggedDTO
	err = mapperWithCustom.MapInto(&dst, TaggedSource{Name: "Grace", Email: "grace@mycorp.example", Age: 31})
	fmt.Printf("mapped=%#v err=%v\n", dst, err)

	fmt.Println("4) per-call custom validator can override instance defaults:")
	another := validator.New()
	_ = another.RegisterValidation("mycorp", func(fl validator.FieldLevel) bool {
		return strings.HasSuffix(strings.ToLower(fl.Field().String()), "@alt.example")
	})
	_, err = mapper.Map[TaggedDTO](
		TaggedSource{Name: "Grace", Email: "grace@mycorp.example", Age: 31},
		mapper.WithTagName("json"),
		mapper.WithValidator(another),
	)
	fmt.Printf("mapped? err=%v\n", err)

	fmt.Println("5) validation error details:")
	_, err = mapper.Map[TaggedDTO](
		TaggedSource{Name: "G", Email: "g@mycorp.example", Age: 10},
		mapper.WithTagName("json"),
		mapper.WithValidator(myCorp),
	)

	if err == nil {
		fmt.Println("expected validation error")
		return
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]string, 0, len(ve))
		for _, fieldErr := range ve {
			fields = append(fields, fmt.Sprintf("%s %s", fieldErr.Field(), fieldErr.Tag()))
		}
		fmt.Println("failed:", strings.Join(fields, ", "))
	} else {
		fmt.Println("error:", err)
	}

	fmt.Println("6) custom validation adapter:")
	_, err = mapper.Map[DTO](
		Source{Name: "Ada", Email: "ada@example.com", Age: 22},
		mapper.WithValidator(mapper.ValidationFunc(func(value any) error {
			dto, ok := value.(*DTO)
			if ok && dto.Name == "Ada" {
				return errors.New("reserved name")
			}
			return nil
		})),
	)
	fmt.Printf("err=%v\n", err)
}
