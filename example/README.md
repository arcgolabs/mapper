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
- `patch-update`: patch-style `MapInto`, `IgnoreNil`, `IgnoreZero`, required
  fields, and default values.
- `dynamic-input`: `map[string]any` input, fallback tags, nested paths, and
  typed mapping errors.
- `binary`: custom protocol payloads via `encoding.BinaryUnmarshaler`.
- `strict-dynamic`: strict top-level key validation for map inputs.
- `field-hooks`: before/after hooks bound to individual destination fields.
- `collection-merge`: merge vs replace strategy for slice/map fields.
- `naming-normalizer`: custom field-name normalizer for legacy or prefixed models.

## Run all examples

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
mapper: $.Value cannot map string to int: invalid number
reusable mapper converter:
ID-7
```

### `hooks`

```text
local before/after hooks:
1 Ada Lovelace mapped
registered after-map hook:
U-3
error before-map hook:
mapper: before-map hook main.Source -> *main.UserDTO failed: id is required
```

### `validation`

```text
1) built-in validation by option:
err=mapper: destination validation failed for main.DTO: Key: 'DTO.Name' Error:Field validation for 'Name' failed on the 'min' tag
2) mapped value with built-in validation:
mapped=main.DTO{Name:"Ada", Email:"ada@example.com", Age:22} err=<nil>
3) register custom validator once and share it on a mapper instance:
mapped=main.TaggedDTO{Name:"Grace", Email:"grace@mycorp.example", Age:31} err=<nil>
4) per-call custom validator can override instance defaults:
mapped? err=mapper: destination validation failed for main.TaggedDTO: Key: 'TaggedDTO.Email' Error:Field validation for 'Email' failed on the 'mycorp' tag
5) validation error details:
failed: Name min, Age gte
6) custom validation adapter:
err=mapper: destination validation failed for main.DTO: reserved name
```

### `instances`

```text
package-level: 2
reusable: 20
```

### `patch-update`

```text
patch: {Name:Grace Email:ada@example.com Age:36 Role:admin} err=<nil>
create: {Name:Lin Role:user Active:true} err=<nil>
```

### `dynamic-input`

```text
dynamic: {ID:7 Name:Ada Email:ada@example.com Role:user} err=<nil>
mapping error: path=$.ID src=[]string dst=int
```

### `binary`

```text
from bytes: main.Packet{Version:1, Kind:"\x01", Payload:[102 111 111]} err=<nil>
from string: main.Packet{Version:1, Kind:"\x01", Payload:[102 111 111]} err=<nil>
from invalid bytes err=mapper: $ [binary] cannot map []uint8 to main.Packet: packet too short
```

### `strict-dynamic`

```text
loose: main.UserDTO{ID:7, Name:"Ada"} err=<nil>
strict unknown keys: [role]
```

### `field-hooks`

```text
dto: main.UserDTO{Name:"ADA LOVELACE", Label:"mapped:admin"}
```

### `collection-merge`

```text
merge: main.Destination{Tags:[]string{"base", "new", "hotfix"}, Meta:map[string]string{"env":"prod", "region":"us-east"}} err=<nil>
replace: main.Destination{Tags:[]string{"again"}, Meta:map[string]string{"team":"infra"}} err=<nil>
```

### `naming-normalizer`

```text
{ID:7 Name:"Ada"}
{ID:42 Name:"Grace Hopper"}
```
