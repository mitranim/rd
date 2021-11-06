## Overview

Short for **R**equest **D**ecoding. Missing feature of the Go standard library: decoding arbitrary HTTP requests into structs. Features:

* Transparent support for different content types / encoding formats:
  * URL query.
  * URL-encoded form.
  * Multipart form.
  * JSON.
* Transparent support for different HTTP methods:
  * Read-only -> parse only URL query.
  * Non-read-only -> parse only request body.
* Transparent support for various text-parsing interfaces.
* Support for membership testing (was X present in request?), useful for PATCH semantics.
* Tiny and dependency-free.

API docs: https://pkg.go.dev/github.com/mitranim/rd.

## Example

1-call decoding. Works for any content type.

```golang
import "github.com/mitranim/rd"
import "github.com/mitranim/try"

var input struct {
  FieldOne string `json:"field_one"`
  FieldTwo int64  `json:"field_two"`
}
try.To(rd.Decode(req, &input))
```

Download once, decode many times. Works for any content type.

```golang
import "github.com/mitranim/rd"
import "github.com/mitranim/try"

dec := rd.TryDownload(req)

var input0 struct {
  FieldOne string `json:"field_one"`
}
try.To(dec.Decode(&input0))

var input1 struct {
  FieldTwo int64  `json:"field_two"`
}
try.To(dec.Decode(&input1))

// Membership testing.
haser := dec.Haser()
fmt.Println(haser.Has(`fieldTwo`))
```

## Changelog

### v0.2.1

Fixed edge case bug where `Form.Decode` wouldn't invoke `SliceParser` for non-slices.

### v0.2.0

Breaking revision:

* Much more flexible.
* Much faster.
* Much more test coverage.
* Renamed from `reqdec` to `rd` for brevity.
* Dependency-free.

### v0.1.7

Breaking: when decoding from JSON, `{"<field>": null}` zeroes the matching destination field, instead of being ignored. This is an intentional deviation. The `json` package makes no distinction between a missing field and a field whose value is `null`. However, in order to support `PATCH` semantics, we often want to decode into _non-zero_ output structs, updating fields that are present in the input, while ignoring fields missing from the input. Using `null` to zero the output is essential for this use case. If `null` was ignored, clients would be unable to set empty values, able only to set new non-empty values.

Breaking: when decoding from formdata, `""` is treated as a "null" or "zero value" for output fields that are not slices and don't implement `SliceParser`. This is extremely useful for standard DOM forms.

Removed `Reqdec.DecodeAt`. It wasn't compatible with the new logic for struct field decoding, and supporting it separately would require code duplication.

Renamed `FromQuery` → `FromVals`.

Added `FromJson`.

### v0.1.6

Added `FromQuery` and `FromReqQuery`.

### v0.1.5

Added `SliceParser` for parsing non-slices from lists.

### v0.1.4

Changed the license to Unlicense.

### v0.1.3

When decoding into a struct where some of the fields are embedded struct pointers, those nested structs are allocated only if some of their fields are present in the request.

Also moved some reflection-related utils to a [tiny dependency](https://github.com/mitranim/refut).

### v0.1.2

First tagged release.

## License

https://unlicense.org

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts
