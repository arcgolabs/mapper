# Performance

`mapper` is reflection-based, so it will not match handwritten assignments or
generated code. The target is a practical middle ground: predictable API,
minimal setup, and cached reflection metadata.

## Hot Path Strategy

- Struct field discovery is cached in mapping plans.
- Each field plan precomputes the built-in operation: assign, convert, pointer,
  struct, slice, map, or dynamic fallback.
- Mapping plans use an external LRU cache rather than a custom cache.
- Converter registries use copy-on-write `collectionx/mapping.Map` snapshots for
  lock-free reads.
- Hook registries use copy-on-write `collectionx/mapping.MultiMap` snapshots for
  lock-free reads.
- Converter-aware field execution checks exact converter keys only for fields
  that have a registered converter snapshot; no-converter mappings skip that path.
- Collection mapping uses `github.com/arcgolabs/collectionx/list` reduce helpers.
- Reflect map iteration uses `MapRange` through a lightweight iterable adapter.

## Current Benchmarks

On the local Windows development machine:

```text
BenchmarkMapWarmPlan-16               ~725 ns/op       288 B/op   10 allocs/op
BenchmarkMapWarmPlanNoConverter-16    ~380-395 ns/op   144 B/op    6 allocs/op
BenchmarkSliceWarmPlan-16            ~2580-2700 ns/op   848 B/op   36 allocs/op
```

The default map and slice benchmarks use a warm plan cache and one converter
for `time.Time` to `string`. The no-converter benchmark shows the lower-cost
field-plan path.

Run benchmarks locally:

```sh
go test -bench . -benchmem
```

## Tradeoffs

Converter lookup remains dynamic because converters can be registered per call.
Structural plans are cached independently from converter sets, which keeps the
cache stable and avoids cache explosion.

Field-level execution plans precompute the built-in path while preserving
converter precedence. A future optimization can add a converter-aware execution
snapshot per mapping context, so fields without possible converters can skip
converter lookup entirely.
