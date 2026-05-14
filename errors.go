package mapper

import (
	"encoding/json"
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

// MappingErrorKind describes the source of a mapping failure.
type MappingErrorKind string

const (
	// KindMapping is a generic mapping failure.
	MappingErrorKindMapping MappingErrorKind = "mapping"
	// KindCannotMap indicates that source and destination kinds/types cannot be
	// combined.
	MappingErrorKindCannotMap MappingErrorKind = "cannot_map"
	// KindConverter indicates a converter execution or return-value failure.
	MappingErrorKindConverter MappingErrorKind = "converter"
	// KindDefault indicates a default-value conversion/parsing failure.
	MappingErrorKindDefaultValue MappingErrorKind = "default_value"
	// KindBinary indicates a binary-unmarshal conversion failure.
	MappingErrorKindBinary MappingErrorKind = "binary"
	// KindHook indicates a field/top-level hook execution failure.
	MappingErrorKindHook MappingErrorKind = "hook"
)

// MappingError describes a path-aware value mapping failure.
type MappingError struct {
	Path            string
	SourceType      reflect.Type
	DestinationType reflect.Type
	Kind            MappingErrorKind
	Cause           error
}

func inferMappingErrorKind(cause error) MappingErrorKind {
	if cause == nil {
		return MappingErrorKindMapping
	}
	if errors.Is(cause, ErrCannotMap) {
		return MappingErrorKindCannotMap
	}
	if errors.Is(cause, ErrDefaultValue) {
		return MappingErrorKindDefaultValue
	}
	return MappingErrorKindMapping
}

func (e *MappingError) Error() string {
	path := e.Path
	if path == "" {
		path = "$"
	}
	if e.Cause == nil {
		return fmt.Sprintf("mapper: %s cannot map %s to %s", path, typeName(e.SourceType), typeName(e.DestinationType))
	}
	if e.Kind == "" || e.Kind == MappingErrorKindMapping {
		return fmt.Sprintf("mapper: %s cannot map %s to %s: %v", path, typeName(e.SourceType), typeName(e.DestinationType), e.Cause)
	}

	return fmt.Sprintf("mapper: %s [%s] cannot map %s to %s: %v", path, e.Kind, typeName(e.SourceType), typeName(e.DestinationType), e.Cause)
}

// Unwrap returns the underlying mapping failure.
func (e *MappingError) Unwrap() error {
	return e.Cause
}

func parseMappingErrorKind(raw string) (MappingErrorKind, error) {
	if raw == "" {
		return MappingErrorKindMapping, nil
	}
	switch MappingErrorKind(raw) {
	case MappingErrorKindMapping,
		MappingErrorKindCannotMap,
		MappingErrorKindConverter,
		MappingErrorKindDefaultValue,
		MappingErrorKindBinary,
		MappingErrorKindHook:
		return MappingErrorKind(raw), nil
	default:
		return MappingErrorKindMapping, fmt.Errorf("mapper: unknown mapping kind %q", raw)
	}
}

func (e *MappingError) MarshalJSON() ([]byte, error) {
	type mappingError struct {
		Path            string           `json:"path"`
		SourceType      string           `json:"sourceType"`
		DestinationType string           `json:"destinationType"`
		Kind            MappingErrorKind `json:"kind"`
		Cause           string           `json:"cause"`
	}
	out := mappingError{
		Path:            e.Path,
		SourceType:      typeName(e.SourceType),
		DestinationType: typeName(e.DestinationType),
		Kind:            e.Kind,
	}
	if e.Cause != nil {
		out.Cause = e.Cause.Error()
	}
	return json.Marshal(out)
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

// UnknownFieldsError is returned when strict dynamic-key mapping detects
// source-map keys that are not used by destination fields.
type UnknownFieldsError struct {
	Source      reflect.Type
	Destination reflect.Type
	Fields      []string
}

func (e *UnknownFieldsError) Error() string {
	return fmt.Sprintf(
		"mapper: %s -> %s strict dynamic mapping encountered unknown keys: %s",
		typeName(e.Source),
		typeName(e.Destination),
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
