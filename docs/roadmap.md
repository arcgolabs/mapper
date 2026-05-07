# Roadmap

## Near Term

- Converter-aware execution snapshots for fields with no possible converter.
- More focused error types for field mapping failures.
- Optional default-value tags.
- Optional required-field tags.

## API Usability

- Add examples for common DTO/entity mapping.
- Add examples for enum and ID converters.
- Add examples for patch/update use cases with `MapInto`.

## Performance

- Avoid converter lookup when no converter snapshot is present.
- Explore cached destination allocation paths for common pointer fields.
- Add benchmark variants for nested structs, slices, and maps.

## Non Goals

- Code generation.
- Expression DSLs in struct tags.
- Implicit converter chains.
- Broad fuzzy matching beyond normalized field names.
