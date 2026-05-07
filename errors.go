package mapper

import (
	"fmt"
	"reflect"
	"strings"
)

// MissingFieldsError is returned in strict mode when destination fields cannot
// be matched to source fields.
type MissingFieldsError struct {
	Source      reflect.Type
	Destination reflect.Type
	Fields      []string
}

func (e *MissingFieldsError) Error() string {
	return fmt.Sprintf(
		"mapper: %s -> %s missing source fields: %s",
		typeName(e.Source),
		typeName(e.Destination),
		strings.Join(e.Fields, ", "),
	)
}

func typeName(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	return t.String()
}
