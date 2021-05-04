## Overview

Tool for decoding an HTTP request into a struct. Transparently supports different content types / encoding formats (JSON, url-encoded form, multipart form). Transparently supports different request methods; for read-only methods such as GET parses inputs from the URL; for non-read-only methods parses inputs _only_ from the body.

## Docs

See the full documentation at https://godoc.org/github.com/mitranim/reqdec.

## Example

```go
dec, err := reqdec.Download(req)
if err != nil {/* ... */}

var input struct {
  FieldOne string `json:"field_one"`
  FieldTwo int64  `json:"field_two"`
}
err = dec.DecodeStruct(&input)
if err != nil {/* ... */}
```

## Changelog

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
