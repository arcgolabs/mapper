package main

import (
	"fmt"

	"github.com/arcgolabs/mapper"
)

func main() {
	type Source struct {
		ID int
	}

	type UserDTO struct {
		ID int
	}

	sliceDst, err := mapper.MapSlice[UserDTO]([]Source{{ID: 1}, {ID: 2}, {ID: 3}})
	if err != nil {
		panic(err)
	}
	fmt.Printf("slice: %+v\n", sliceDst)

	mapDst, err := mapper.MapMap[UserDTO](
		map[string]Source{
			"first":  {ID: 10},
			"second": {ID: 20},
		},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("map:", mapDst["first"].ID, mapDst["second"].ID)
}
