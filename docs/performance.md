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
- Reused per-context execution snapshots are stored in `collectionx/mapping.MultiMap`
  so a struct plan only pays converter matching once when the same plan is used
  repeatedly in a mapping context.
- Dynamic `map[string]any` -> struct mapping can also enforce strict top-level
  key checks with `WithStrictDynamicMapKeys(true)`, using `collectionx/mapping`
  lookups for unknown-key detection.
- Collection mapping uses `github.com/arcgolabs/collectionx/list` reduce helpers.
- Reflect map iteration uses `MapRange` through a lightweight iterable adapter.
- Slice and map element mapping precomputes element operations before iterating.

## Current Benchmarks

On the local Windows development machine:

```text
BenchmarkMapWarmPlan-16               ~740-800 ns/op   336 B/op   10 allocs/op
BenchmarkMapWarmPlanNoConverter-16    ~395-440 ns/op   192 B/op    6 allocs/op
BenchmarkSliceWarmPlan-16            ~2900-3300 ns/op  2882 B/op   41 allocs/op
BenchmarkNestedStructWarmPlan-16      ~280 ns/op       176 B/op    4 allocs/op
BenchmarkLargeSliceWarmPlanNoConverter-16
                                      ~36 us/op       13.7 KB/op 649 allocs/op
BenchmarkMapStringAnyToStruct-16     ~2.0-2.2 us/op   3.4 KB/op  40 allocs/op
```

The default map and slice benchmarks use a warm plan cache and one converter
for `time.Time` to `string`. The no-converter benchmark shows the lower-cost
field-plan path.

Run benchmarks locally:

```sh
go test -bench . -benchmem
```

The benchmark suite also includes nested structs, large slices, dynamic
`map[string]any` input, converter-heavy mapping, and validation-on mapping.

## Tradeoffs

Converter selection remains context-local because converters can be registered
per call. Structural plans are cached independently from converter sets, which
keeps the cache stable and avoids cache explosion.

Field-level execution plans precompute the built-in path while preserving
converter precedence. Dynamic `interface{}` paths still use runtime lookup
because the concrete source type is not known when the structural plan is built.
