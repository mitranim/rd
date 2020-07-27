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

## License

https://en.wikipedia.org/wiki/WTFPL

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts
