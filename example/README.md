# Example Scenarios

Each folder below is an independent `package main` example that can be run separately.

- `basic`: direct mapping with `mapper.Map`.
- `field-mapping`: `mapper` tags, nested source paths, skip fields, and field
  naming.
- `collections`: `MapSlice` and `MapMap`.
- `converters`: inline converters, error converters, and reusable converter
  registration.
- `hooks`: before/after hooks, error hooks, and hook registration.
- `validation`: destination validation by option and reusable mapper instances.
- `instances`: package-level API vs reusable mapper instances.

## Run all examples

```sh
go run ./example/basic
go run ./example/field-mapping
go run ./example/collections
go run ./example/converters
go run ./example/hooks
go run ./example/validation
go run ./example/instances
```

## Expected outputs

### `basic`

```text
main.UserDTO{ID:1, Name:"Ada"}
```

### `field-mapping`

```text
7 Grace Hopper 155-9001
```

### `collections`

```text
slice: [{ID:1} {ID:2} {ID:3}]
map: 10 20
```

### `converters`

```text
converter by option:
2026-05-07T10:30:00Z
error converter:
invalid number
reusable mapper converter:
ID-7
```

### `hooks`

```text
local before/after hooks:
10 Ada Lovelace mapped
registered after-map hook:
U-3
error before-map hook:
id is required
```

### `validation`

```text
1) built-in validation by option:
mapped? err=mapper: destination validation failed: Key: 'DTO.Name' Error:Field validation for 'Name' failed on the 'min' tag
2) mapped value with built-in validation:
mapped=main.DTO{Name:"Ada", Email:"ada@example.com", Age:22} err=<nil>
3) register custom validator once and share it on a mapper instance:
mapped=main.TaggedDTO{Name:"Grace", Email:"grace@mycorp.example", Age:31} err=<nil>
4) per-call custom validator can override instance defaults:
mapped? err=mapper: destination validation failed: Key: 'TaggedDTO.Email' Error:Field validation for 'Email' failed on the 'mycorp' tag
5) validation error details:
failed: Name min, Age gte
```

### `instances`

```text
package-level: 2
reusable: 20
```
