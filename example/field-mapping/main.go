package main

import (
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type Profile struct {
		Name string
	}
	type Source struct {
		UserID  int
		Display string
		Meta    *struct {
			Profile *Profile
		}
	}

	type UserDTO struct {
		ID      int    `mapper:"user_id"`
		Label   string `mapper:"display"`
		PhoneNo string `mapper:"meta.profile.name"`
		Ignore  string `mapper:"-"`
	}

	dto, err := mapper.Map[UserDTO](
		Source{
			UserID:  7,
			Display: "Grace Hopper",
			Meta: &struct {
				Profile *Profile
			}{
				Profile: &Profile{Name: "155-9001"},
			},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(dto.ID, dto.Label, dto.PhoneNo, dto.Ignore)
}
