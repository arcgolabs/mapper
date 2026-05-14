package main

import "fmt"

import "github.com/arcgolabs/mapper"

func main() {
	type Source struct {
		Tags []string
		Meta map[string]string
	}

	type Destination struct {
		Tags []string
		Meta map[string]string
	}

	dest := Destination{
		Tags: []string{"base"},
		Meta: map[string]string{
			"env": "dev",
		},
	}

	err := mapper.MapInto(&dest, Source{
		Tags: []string{"new", "hotfix"},
		Meta: map[string]string{
			"region": "us-east",
			"env":    "prod",
		},
	}, mapper.UpdateMergeMode())

	fmt.Printf("merge: %#v err=%v\n", dest, err)

	err = mapper.MapInto(&dest, Source{
		Tags: []string{"again"},
		Meta: map[string]string{"team": "infra"},
	}, mapper.UpdateReplaceMode())

	fmt.Printf("replace: %#v err=%v\n", dest, err)
}
