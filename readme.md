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

### v0.1.3

When decoding into a struct where some of the fields are embedded struct pointers, those nested structs are allocated only if some of their fields are present in the request.

Also moved some reflection-related utils to a [tiny dependency](https://github.com/mitranim/refut).

### v0.1.2

First tagged release.

## License

https://en.wikipedia.org/wiki/WTFPL

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts
