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
2. Apply an exact converter when one is registered.
3. Use the precompiled built-in operation when possible.
4. Recurse into pointers, slices, maps, and structs.
5. Return a path-aware error when the value cannot be mapped.

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

Nested source paths are supported in tags:

```go
type UserDTO struct {
	Name string `mapper:"profile.name"`
}
```

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

## Hooks

Hooks are exact type-pair functions:

```go
func(S, *D)
func(S, *D) error
```

Hooks run only for the top-level mapping call. They do not run for every nested
struct or collection item. This keeps hook behavior predictable and avoids
surprising side effects in deep object graphs.

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
