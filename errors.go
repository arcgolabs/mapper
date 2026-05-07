package mapper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	// ErrCannotMap is wrapped by MappingError when no built-in mapping path or
	// converter can handle a source/destination type pair.
	ErrCannotMap = errors.New("mapper: cannot map value")

	// ErrDefaultValue is wrapped by MappingError when a default tag value cannot
	// be assigned to the destination field.
	ErrDefaultValue = errors.New("mapper: invalid default value")
)

// MappingError describes a path-aware value mapping failure.
type MappingError struct {
	Path            string
	SourceType      reflect.Type
	DestinationType reflect.Type
	Cause           error
}

func (e *MappingError) Error() string {
	path := e.Path
	if path == "" {
		path = "$"
	}
	if e.Cause == nil {
		return fmt.Sprintf("mapper: %s cannot map %s to %s", path, typeName(e.SourceType), typeName(e.DestinationType))
	}
	return fmt.Sprintf("mapper: %s cannot map %s to %s: %v", path, typeName(e.SourceType), typeName(e.DestinationType), e.Cause)
}

// Unwrap returns the underlying mapping failure.
func (e *MappingError) Unwrap() error {
	return e.Cause
}

// MissingFieldsError is returned when strict mode or required tags detect
// destination fields that cannot be matched to source fields.
type MissingFieldsError struct {
	Source      reflect.Type
	Destination reflect.Type
	Fields      []string
	Required    bool
}

func (e *MissingFieldsError) Error() string {
	kind := "missing source fields"
	if e.Required {
		kind = "missing required source fields"
	}
	return fmt.Sprintf(
		"mapper: %s -> %s %s: %s",
		typeName(e.Source),
		typeName(e.Destination),
		kind,
		strings.Join(e.Fields, ", "),
	)
}

// ValidationError wraps failures returned by ValidationEngine.
type ValidationError struct {
	ValueType reflect.Type
	Cause     error
}

func (e *ValidationError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("mapper: destination validation failed for %s", typeName(e.ValueType))
	}
	return fmt.Sprintf("mapper: destination validation failed for %s: %v", typeName(e.ValueType), e.Cause)
}

// Unwrap returns the underlying validation failure.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

type panicError struct {
	value any
}

func (e panicError) Error() string {
	return fmt.Sprintf("panic: %v", e.value)
}

func typeName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	return t.String()
}
