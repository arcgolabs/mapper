package mapper_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/arcgolabs/mapper"
)

func TestMapSameNames(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}
	type userDTO struct {
		ID   int
		Name string
	}

	dto, err := mapper.Map[userDTO](user{ID: 12, Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	if dto.ID != 12 || dto.Name != "Ada" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestMapTagAndSkip(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}
	type userDTO struct {
		UserID int    `mapper:"id"`
		Label  string `mapper:"name"`
		Skip   string `mapper:"-"`
	}

	dto, err := mapper.Map[userDTO](user{ID: 7, Name: "Grace"})
	if err != nil {
		t.Fatal(err)
	}

	if dto.UserID != 7 || dto.Label != "Grace" || dto.Skip != "" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestMapNormalizesNames(t *testing.T) {
	type user struct {
		UserID int
	}
	type userDTO struct {
		UserID int `mapper:"user_id"`
	}

	dto, err := mapper.Map[userDTO](user{UserID: 99})
	if err != nil {
		t.Fatal(err)
	}

	if dto.UserID != 99 {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestMapNestedPath(t *testing.T) {
	type profile struct {
		Name string
	}
	type user struct {
		Profile *profile
	}
	type userDTO struct {
		Name string `mapper:"profile.name"`
	}

	dto, err := mapper.Map[userDTO](user{Profile: &profile{Name: "Lin"}})
	if err != nil {
		t.Fatal(err)
	}

	if dto.Name != "Lin" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestMapPointerAndSlice(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID int
	}

	src := []*user{{ID: 1}, {ID: 2}}
	dst, err := mapper.Slice[*userDTO](src)
	if err != nil {
		t.Fatal(err)
	}

	if len(dst) != 2 || dst[0] == nil || dst[1] == nil || dst[0].ID != 1 || dst[1].ID != 2 {
		t.Fatalf("unexpected dst: %+v", dst)
	}
}

func TestMapSliceHelper(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID int
	}

	dst, err := mapper.MapSlice[userDTO]([]user{{ID: 1}, {ID: 2}})
	if err != nil {
		t.Fatal(err)
	}

	if len(dst) != 2 || dst[0].ID != 1 || dst[1].ID != 2 {
		t.Fatalf("unexpected dst: %+v", dst)
	}
}

func TestMapMapValues(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID int
	}

	dst, err := mapper.Map[map[string]userDTO](map[string]user{
		"first":  {ID: 1},
		"second": {ID: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	if dst["first"].ID != 1 || dst["second"].ID != 2 {
		t.Fatalf("unexpected dst: %+v", dst)
	}
}

func TestMapMapHelper(t *testing.T) {
	type user struct {
		ID int
	}
	type userDTO struct {
		ID int
	}

	dst, err := mapper.MapMap[userDTO](map[string]user{
		"first":  {ID: 1},
		"second": {ID: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	if dst["first"].ID != 1 || dst["second"].ID != 2 {
		t.Fatalf("unexpected dst: %+v", dst)
	}
}

func TestStrictMissingFields(t *testing.T) {
	type src struct {
		ID int
	}
	type dst struct {
		ID   int
		Name string
	}

	_, err := mapper.Map[dst](src{ID: 1}, mapper.Strict())
	if err == nil {
		t.Fatal("expected strict mode error")
	}

	var missing *mapper.MissingFieldsError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingFieldsError, got %T: %v", err, err)
	}
	if !reflect.DeepEqual(missing.Fields, []string{"Name"}) {
		t.Fatalf("unexpected missing fields: %+v", missing.Fields)
	}
}

func TestMapIntoPreservesMissingFieldsWithoutStrict(t *testing.T) {
	type src struct {
		ID int
	}
	type dst struct {
		ID   int
		Name string
	}

	value := dst{Name: "existing"}
	if err := mapper.MapInto(&value, src{ID: 10}); err != nil {
		t.Fatal(err)
	}

	if value.ID != 10 || value.Name != "existing" {
		t.Fatalf("unexpected dst: %+v", value)
	}
}
