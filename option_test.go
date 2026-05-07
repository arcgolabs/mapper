package mapper_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/arcgolabs/mapper"
)

func TestIgnoreNilAndIgnoreZeroPreserveDestination(t *testing.T) {
	type patch struct {
		Name *string
		Age  int
	}
	type user struct {
		Name string
		Age  int
	}

	dst := user{Name: "Ada", Age: 36}
	if err := mapper.MapInto(&dst, patch{}, mapper.IgnoreNil(), mapper.IgnoreZero()); err != nil {
		t.Fatal(err)
	}

	if dst.Name != "Ada" || dst.Age != 36 {
		t.Fatalf("unexpected dst: %+v", dst)
	}
}

func TestRequiredTagReportsMissingSource(t *testing.T) {
	type src struct {
		ID int
	}
	type dst struct {
		ID   int
		Name string `mapper:",required"`
	}

	_, err := mapper.Map[dst](src{ID: 1})
	var missing *mapper.MissingFieldsError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingFieldsError, got %T: %v", err, err)
	}
	if !missing.Required || !reflect.DeepEqual(missing.Fields, []string{"Name"}) {
		t.Fatalf("unexpected missing fields: %+v required=%v", missing.Fields, missing.Required)
	}
}

func TestDefaultTagFillsMissingAndZeroValues(t *testing.T) {
	type src struct {
		Name string
	}
	type dst struct {
		Name string `mapper:",default=anonymous"`
		Age  int    `mapper:",default=18"`
	}

	value, err := mapper.Map[dst](src{})
	if err != nil {
		t.Fatal(err)
	}
	if value.Name != "anonymous" || value.Age != 18 {
		t.Fatalf("unexpected dst: %+v", value)
	}
}

func TestFallbackTagsUseJSONFieldNames(t *testing.T) {
	type src struct {
		UserID int `json:"user_id"`
	}
	type dst struct {
		UserID int `json:"user_id"`
	}

	value, err := mapper.Map[dst](src{UserID: 42}, mapper.WithFallbackTags("json"))
	if err != nil {
		t.Fatal(err)
	}
	if value.UserID != 42 {
		t.Fatalf("unexpected dst: %+v", value)
	}
}

func TestMapStringAnyToStruct(t *testing.T) {
	type dst struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `mapper:"profile.email"`
	}

	value, err := mapper.Map[dst](map[string]any{
		"id":   float64(12),
		"name": "Ada",
		"profile": map[string]any{
			"email": "ada@example.com",
		},
	}, mapper.WithFallbackTags("json"))
	if err != nil {
		t.Fatal(err)
	}
	if value.ID != 12 || value.Name != "Ada" || value.Email != "ada@example.com" {
		t.Fatalf("unexpected dst: %+v", value)
	}
}

func TestMapStringAnyNestedPathToStruct(t *testing.T) {
	type dst struct {
		Email string `mapper:"profile.email"`
	}

	value, err := mapper.Map[dst](map[string]any{
		"profile": map[string]any{
			"email": "ada@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Email != "ada@example.com" {
		t.Fatalf("unexpected dst: %+v", value)
	}
}

func TestMappingErrorWrapsConverterFailure(t *testing.T) {
	type src struct {
		Value string
	}
	type dst struct {
		Value int
	}

	sentinel := errors.New("bad value")
	_, err := mapper.Map[dst](src{Value: "x"}, mapper.ConverterE(func(string) (int, error) {
		return 0, sentinel
	}))
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}

	var mappingErr *mapper.MappingError
	if !errors.As(err, &mappingErr) {
		t.Fatalf("expected MappingError, got %T: %v", err, err)
	}
	if mappingErr.Path != "$.Value" {
		t.Fatalf("unexpected path: %s", mappingErr.Path)
	}
}

func TestValidationFuncWrapsValidationError(t *testing.T) {
	type user struct {
		Name string
	}

	sentinel := errors.New("invalid")
	_, err := mapper.Map[user](user{Name: "Ada"}, mapper.WithValidator(mapper.ValidationFunc(func(any) error {
		return sentinel
	})))
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected validation sentinel, got %v", err)
	}

	var validationErr *mapper.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}
