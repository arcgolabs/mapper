package mapper_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arcgolabs/mapper"
)

func TestMapWithLocalConverter(t *testing.T) {
	type event struct {
		At time.Time
	}
	type eventDTO struct {
		At string
	}

	at := time.Date(2026, 5, 7, 10, 30, 0, 0, time.UTC)
	dto, err := mapper.Map[eventDTO](
		event{At: at},
		mapper.Converter(func(v time.Time) string {
			return v.Format(time.RFC3339)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if dto.At != "2026-05-07T10:30:00Z" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestConverterOverridesAssignableField(t *testing.T) {
	type user struct {
		Name string
	}
	type userDTO struct {
		Name string
	}

	dto, err := mapper.Map[userDTO](
		user{Name: "Ada"},
		mapper.Converter(func(v string) string {
			return "converted:" + v
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if dto.Name != "converted:Ada" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

func TestMapWithErrorConverter(t *testing.T) {
	type src struct {
		Value string
	}
	type dst struct {
		Value int
	}

	sentinel := errors.New("invalid value")
	_, err := mapper.Map[dst](
		src{Value: "bad"},
		mapper.ConverterE(func(v string) (int, error) {
			return 0, sentinel
		}),
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected converter error, got %v", err)
	}
}

func TestMapperRegisterConverter(t *testing.T) {
	type customID int
	type user struct {
		ID customID
	}
	type userDTO struct {
		ID string
	}

	m := mapper.New()
	if err := m.Register(func(v customID) string {
		return fmt.Sprintf("U-%d", v)
	}); err != nil {
		t.Fatal(err)
	}

	var dto userDTO
	if err := m.MapInto(&dto, user{ID: 42}); err != nil {
		t.Fatal(err)
	}

	if dto.ID != "U-42" {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}
