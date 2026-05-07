# mapper

`mapper` is a small Go mapping library inspired by Java MapStruct, but without
code generation. It keeps call sites generic and type-oriented while using
cached, lightweight reflection internally.

## Install

```sh
go get github.com/arcgolabs/mapper
```

## Quick Start

```go
type User struct {
	ID   int
	Name string
}

type UserDTO struct {
	ID   int
	Name string
}

dto, err := mapper.Map[UserDTO](User{ID: 1, Name: "Ada"})
```

The generic helpers keep the destination type visible at the call site:

```go
dto, err := mapper.Map[UserDTO](user)
err := mapper.MapInto(&dto, user)
dtos, err := mapper.MapSlice[UserDTO](users)
dtoByID, err := mapper.MapMap[UserDTO](usersByID)
```

### Examples

The repository uses a dedicated `example/` directory for runnable usage scenarios.
See [examples index](example/README.md).

```sh
go run ./example/basic
go run ./example/field-mapping
go run ./example/collections
go run ./example/converters
go run ./example/hooks
go run ./example/validation
go run ./example/instances
```

`MapSlice` and `MapMap` infer the source type from the argument, so only the
destination element/value type needs to be written.

## Field Matching

Fields are matched by normalized names. Case, underscores, hyphens, spaces, and
dots are ignored, so `UserID`, `user_id`, and `user-id` all match.

Use `mapper` tags on destination fields when names differ:

```go
type UserDTO struct {
	UserID int    `mapper:"id"`
	Label  string `mapper:"name"`
	Skip   string `mapper:"-"`
}
```

Nested source paths are supported:

```go
type UserDTO struct {
	Name string `mapper:"profile.name"`
}
```

Change the tag name when integrating with an existing model:

```go
m := mapper.New(mapper.WithTagName("map"))
```

## Converters

Converters run before built-in assignment and conversion. Use them for business
rules such as IDs, timestamps, enums, and formatting.

```go
dto, err := mapper.Map[EventDTO](
	event,
	mapper.Converter(func(v time.Time) string {
		return v.Format(time.RFC3339)
	}),
)
```

Error-returning converters are supported:

```go
mapper.ConverterE(func(v string) (UserID, error) {
	return ParseUserID(v)
})
```

Register converters on a `Mapper` instance to reuse them:

```go
m := mapper.New()
_ = m.Register(func(v CustomID) string {
	return fmt.Sprintf("U-%d", v)
})

var dto UserDTO
err := m.MapInto(&dto, user)
```

## Hooks

Use hooks for small pieces of mapping logic that should stay handwritten. Hooks
run only around the top-level mapping call and match the exact source type plus
destination pointer type.

```go
m := mapper.New(
	mapper.AfterMap(func(src User, dst *UserDTO) {
		dst.FullName = src.FirstName + " " + src.LastName
	}),
)
```

Use `BeforeMapE` or `AfterMapE` when the hook can fail:

```go
m := mapper.New(
	mapper.BeforeMapE(func(src User, dst *UserDTO) error {
		if src.ID == 0 {
			return errors.New("missing user id")
		}
		return nil
	}),
)
```

## Validation

`mapper` can validate the mapped destination through any type that implements:

```go
type ValidationEngine interface {
	Struct(any) error
}
```

This is intentionally small so you can plug in standard validator implementations
or custom ones with your own rules:

```go
import "github.com/go-playground/validator/v10"

validate := validator.New()
dto, err := mapper.Map[UserDTO](source, mapper.WithValidator(validate))
```

You can also store the validator on a reusable `Mapper` instance:

```go
m := mapper.New(
	mapper.WithTagName("json"),
	mapper.WithValidator(validate),
)
_ = m.MapInto(&dto, source)
```

## Strict Mode

By default, unmatched destination fields are left unchanged. Use strict mode to
turn those into errors.

```go
dto, err := mapper.Map[UserDTO](user, mapper.Strict())
```

## Taskfile

This repository includes a `Taskfile.yml` for reproducible local workflows:

```sh
# Quality checks
task preflight

# Run all examples
task examples

# Create release tag
task release VERSION=v0.1.0
task release VERSION=v0.1.0 PUSH=true
```

## Cache And Performance

Mapping plans are cached with `github.com/hashicorp/golang-lru/v2`. The default
cache size is 1024 type pairs.

```go
m := mapper.New(mapper.WithPlanCacheSize(4096))
```

The implementation also uses `github.com/arcgolabs/collectionx` submodules for
collection helpers and keeps converter/hook registries as copy-on-write
snapshots for lock-free reads during mapping.
The plan cache stays on `github.com/hashicorp/golang-lru/v2` for bounded LRU
eviction.

## Behavior Summary

- Destination fields are the mapping target; source-only fields are ignored.
- Exported fields only are mapped.
- `mapper:"-"` skips a destination field.
- Converters take precedence over built-in assignment and conversion.
- Hooks run for the top-level call only, not for nested fields or collection items.
- Nil source pointers, maps, and slices map to zero values.
- Whole struct, slice, and map conversion is avoided so nested mapping and
  converters can run field-by-field.
- `MapInto` preserves unmatched destination fields unless strict mode is enabled.

## More

- [Design](docs/design.md)
- [Performance](docs/performance.md)
- [Roadmap](docs/roadmap.md)
```

## Cache

Mapping plans are cached with `github.com/hashicorp/golang-lru/v2`. The default
cache size is 1024 type pairs.

```go
m := mapper.New(mapper.WithPlanCacheSize(4096))
```
