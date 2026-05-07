package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/arcgolabs/mapper"
)

func main() {
	fmt.Println("converter by option:")

	type Source struct {
		At time.Time
	}
	type DTO struct {
		At string
	}

	dto, err := mapper.Map[DTO](
		Source{At: time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)},
		mapper.Converter(func(v time.Time) string {
			return v.Format(time.RFC3339)
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(dto.At)

	fmt.Println("error converter:")

	type NumSource struct {
		Value string
	}
	type NumDTO struct {
		Value int
	}

	_, err = mapper.Map[NumDTO](
		NumSource{Value: "bad-number"},
		mapper.ConverterE(func(v string) (int, error) {
			return 0, errors.New("invalid number")
		}),
	)
	fmt.Println(err)

	fmt.Println("reusable mapper converter:")

	type IDSource struct {
		ID int
	}
	type IDDTO struct {
		ID string
	}

	m := mapper.New()
	if err := m.Register(func(v int) string {
		return "ID-" + strconv.Itoa(v)
	}); err != nil {
		panic(err)
	}
	var dst IDDTO
	if err := m.MapInto(&dst, IDSource{ID: 7}); err != nil {
		panic(err)
	}
	fmt.Println(dst.ID)
}
