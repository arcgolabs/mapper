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
go run ./example/patch-update
go run ./example/dynamic-input
go run ./example/binary
go run ./example/strict-dynamic
go run ./example/field-hooks
go run ./example/collection-merge
go run ./example/naming-normalizer
```

`MapSlice` and `MapMap` infer the source type from the argument, so only the
destination element/value type needs to be written.

## Field Matching

Fields are matched by normalized names. Case, underscores, hyphens, spaces, and
dots are ignored, so `UserID`, `user_id`, and `user-id` all match.

If you have legacy naming schemes, you can replace the normalizer:

```go
normalizeLegacy := func(name string) string {
    normalized := strings.ToLower(strings.TrimSpace(name))
    normalized = strings.TrimPrefix(normalized, "u_")
    return strings.NewReplacer("-", "", "_", "", " ", "", ".", "").Replace(normalized)
}

dto, err := mapper.Map[UserDTO](
    LegacySource{},
    mapper.WithNameNormalizer(normalizeLegacy),
)
```

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

Use fallback tags when models already carry tags such as `json` or `yaml`:

```go
dto, err := mapper.Map[UserDTO](input, mapper.WithFallbackTags("json", "yaml"))
```

Destination `mapper` tags can also declare required fields and simple defaults:

```go
type UserDTO struct {
	Name string `mapper:",required"`
	Role string `mapper:",default=user"`
}
```

`map[string]any` sources can be mapped into structs. This is useful for decoded
configuration, JSON-like data, and private protocol payloads that first land in
a dynamic map:

```go
dto, err := mapper.Map[UserDTO](
	map[string]any{"id": 7, "name": "Ada"},
	mapper.WithFallbackTags("json"),
)
```

For custom protocol formats, destination structs can implement
`encoding.BinaryUnmarshaler` and map directly from `[]byte` or string payloads:

```go
type Packet struct {
	...
}

dto, err := mapper.Map[Packet]([]byte{0x01, 0x02, 0x03})
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

Use hooks for small pieces of mapping logic that should stay handwritten. Top-level
hooks run around each mapping call and match the exact source type plus
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

Field hooks run before/after a specific destination field assignment.

```go
m := mapper.New(
	mapper.BeforeField("DisplayName", func(src User, dst *UserDTO, field *string) {
		*field = src.FirstName + " " + src.LastName
	}),
	mapper.AfterField("Label", func(src User, dst *UserDTO, field *string) error {
		*field = "mapped:" + src.Role
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

For small custom validators, use `ValidationFunc`:

```go
err := mapper.MapInto(&dto, source, mapper.WithValidator(mapper.ValidationFunc(func(v any) error {
	return nil
})))
```

## Strict Mode

By default, unmatched destination fields are left unchanged. Use strict mode to
turn those into errors.

```go
dto, err := mapper.Map[UserDTO](user, mapper.Strict())
```

For `map[string]any` inputs, use `WithStrictDynamicMapKeys(true)` to fail when
incoming top-level keys are not bound to any destination field:

```go
dto, err := mapper.Map[UserDTO](
	map[string]any{"id": 1, "name": "Ada", "extra": "ignored"},
	mapper.WithStrictDynamicMapKeys(true),
)
```

## Patch Updates

`MapInto` can be used for patch/update workflows. `IgnoreNil` and `IgnoreZero`
leave the existing destination value untouched for nil or zero source values:

```go
err := mapper.MapInto(&entity, patch, mapper.IgnoreNil(), mapper.IgnoreZero())
```

You can also control collection/map update behavior:

```go
err := mapper.MapInto(&entity, patch, mapper.UpdateMergeMode())
err = mapper.MapInto(&entity, patch, mapper.UpdateReplaceMode())
```

## Errors

Field-level mapping failures wrap `MappingError`, which carries the field path
and source/destination types. Validation failures wrap `ValidationError`.

```go
var mappingErr *mapper.MappingError
if errors.As(err, &mappingErr) {
	fmt.Println(mappingErr.Path)
}
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
- `mapper:",required"` requires a matching source field.
- `mapper:",default=value"` fills missing or zero source values.
- Converters take precedence over built-in assignment and conversion.
- Hooks include both top-level call hooks and destination field hooks.
- Nil source pointers, maps, and slices map to zero values.
- `UpdateMergeMode()` appends/replaces collection fields by merge; `UpdateReplaceMode()`
  resets them.
- `IgnoreNil` and `IgnoreZero` preserve destination values in patch-style calls.
- Whole struct, slice, and map conversion is avoided so nested mapping and
  converters can run field-by-field.
- `MapInto` preserves unmatched destination fields unless strict mode is enabled.

## More

- [Design](docs/design.md)
- [Performance](docs/performance.md)
- [Roadmap](docs/roadmap.md)
