# Design

`mapper` is intentionally smaller than Java MapStruct. It does not generate
code, parse expressions, or try to emulate compile-time validation. The library
focuses on predictable runtime mapping with generic entry points.

## Principles

- Keep call sites explicit about the destination type.
- Build reflection metadata once and reuse it through an LRU plan cache.
- Prefer exact and simple rules over expression languages.
- Allow handwritten business logic through converters and hooks.
- Keep core behavior independent from global mutable state where possible.

## Mapping Pipeline

Top-level mapping runs in this order:

1. Build a mapping context from the `Mapper` and per-call options.
2. Run matching `BeforeMap` hooks.
3. Map the source value into the destination value.
4. Run matching `AfterMap` hooks.

Field mapping runs in this order:

1. Resolve source and destination field values from the cached plan.
2. Apply a default tag value when the source value is missing or zero.
3. Run destination field-level `BeforeField` hooks.
4. Skip nil or zero source values when patch-style options request it.
5. Apply an exact converter when one is registered.
6. If the destination implements `encoding.BinaryUnmarshaler`, invoke
   `UnmarshalBinary` for `[]byte` / string payloads.
7. Use the precompiled built-in operation when possible.
8. Recurse into pointers, slices, maps, and structs.
9. Run destination field-level `AfterField` hooks.
10. Return a path-aware error when the value cannot be mapped.

## Field Matching

The destination type drives mapping. Each exported destination field is matched
against exported source fields by normalized name. Normalization removes `_`,
`-`, spaces, and `.`, then lowercases the rest.

Destination tags override the source name:

```go
type UserDTO struct {
	UserID int `mapper:"id"`
	Name   string
}
```

Tags can also express required source fields and simple defaults:

```go
type UserDTO struct {
	Name string `mapper:",required"`
	Role string `mapper:",default=user"`
}
```

Nested source paths are supported in tags:

```go
type UserDTO struct {
	Name string `mapper:"profile.name"`
}
```

Fallback tags can be enabled when existing models already carry field names:

```go
dto, err := mapper.Map[UserDTO](input, mapper.WithFallbackTags("json", "yaml"))
```

When the source is `map[string]any` and strict dynamic key mode is enabled, mapping
tracks top-level keys used by destination field bindings and reports unknown keys
via `UnknownFieldsError`.

## Converters

Converters are exact type-pair functions:

```go
func(S) D
func(S) (D, error)
```

They are registered either per call or on a `Mapper`. Converter lookup uses a
copy-on-write registry backed by `collectionx/mapping.Map`, so writes are
synchronized and reads use immutable snapshots during mapping.

The converter snapshot uses the non-concurrent `mapping.Map` deliberately. Once
a mapping context is built, that snapshot is immutable; concurrent writes create
a new snapshot instead of mutating the existing one.

Converters intentionally match exact source and destination types. This keeps
selection predictable and avoids implicit converter chains.

When the same struct plan is reused inside a mapping context, such as slice item
mapping, the context stores a converter-aware execution snapshot in
`collectionx/mapping.MultiMap`. Fields with an exact converter call it directly;
fields without a possible converter skip converter map lookup and use the
precompiled operation. Single top-level struct mapping keeps the cheaper direct
lookup path to avoid cache allocation overhead.

## Hooks

Top-level and field hooks are exact type-based functions.

```go
func(S, *D)
func(S, *D) error
func(S, *D, *F)
func(S, *D, *F) error
```

Top-level hooks run for the root mapping call only. Field hooks run during field
assignment in the destination mapping pipeline and can be triggered for nested
objects.

Hook state belongs to a `Mapper` instance. Package-level hook registration uses
the package default mapper; there is no separate package-global hook map. Hook
storage is backed by `collectionx/mapping.MultiMap` because each source and
destination type pair can have multiple hooks.

Hook snapshots follow the same copy-on-write rule as converters, so they do not
need `ConcurrentMap` for read safety.

## Caching

Mapping plans are cached by source type, destination type, and tag name. The
cache is an LRU cache from `github.com/hashicorp/golang-lru/v2`.

Plans are independent from converters and hooks. That is deliberate: per-call
converters and hooks can change without invalidating structural field plans.

Each field step precomputes the built-in operation for its static source and
destination field types. Converter lookup remains runtime data, so converters
still take precedence over assignment and conversion.

Non-cache internal lookup structures use `collectionx/mapping` containers for
consistency across the codebase. The plan cache remains a dedicated LRU because
it needs bounded eviction behavior.

## Dynamic Map Sources

`map[string]any` sources are supported for struct destinations. Map keys are
matched using the same normalization rules as struct fields, and destination
tags plus fallback tags are honored. Nested source paths can traverse nested
maps or structs.

This is intended for decoded dynamic input such as configuration, JSON-like
payloads, and private protocol messages that have already been parsed into a
map representation.

## Patch Updates

`IgnoreNil` and `IgnoreZero` change mapping into patch/update semantics by
leaving existing destination values unchanged when the source value should be
treated as absent. Explicit default tags take precedence over ignore options.

## Errors And Validation

Value mapping failures return `MappingError` with a field path and source /
destination types. Validation failures return `ValidationError` while preserving
the original validator error through `Unwrap`.
