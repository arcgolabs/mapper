# Roadmap

## Near Term

- Optional field-level naming strategies beyond normalized names.
- Broader default parsing for collection and struct literals.
- More examples for update/patch workflows with real persistence models.
- Optional strict handling for unknown dynamic map keys.

## API Usability

- Add examples for common DTO/entity mapping.
- Add examples for enum and ID converters.
- Add examples for dynamic input from decoded protocol or configuration maps.

## Performance

- Explore cached destination allocation paths for common pointer fields.
- Profile dynamic `map[string]any` mapping and nested source-path lookup.
- Explore lower-allocation collection mapping for large slices.

## Non Goals

- Code generation.
- Expression DSLs in struct tags.
- Implicit converter chains.
- Broad fuzzy matching beyond normalized field names.
