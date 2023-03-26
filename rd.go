package rd

import (
	"encoding/json"
	"net/http"
)

const (
	Type = `Content-Type`

	TypeJson     = `application/json`
	TypeJsonUtf8 = `application/json; charset=utf-8`

	TypeForm     = `application/x-www-form-urlencoded`
	TypeFormUtf8 = `application/x-www-form-urlencoded; charset=utf-8`

	TypeMulti     = `multipart/form-data`
	TypeMultiUtf8 = `multipart/form-data; charset=utf-8`

	// Used for `(*Request).ParseMultipartForm`.
	// 32 MB, same as the default in the "http" package.
	BufSize = 32 << 20
)

// Returned by `rd.Download`. Implemented by all decoder types in this package.
type Dec interface {
	Decoder
	Haserer
	Setter
}

/*
Short for "decoder". Represents a stored request body and decodes that body into
Go structures. The exact representation and the supported output depends on the
request:

	* GET request          -> backed by `url.Values`, decodes into structs.
	* Form-encoded request -> backed by `url.Values`, decodes into structs.
	* Multipart request    -> backed by `url.Values`, decodes into structs.
	* JSON request         -> backed by `[]byte`, decodes into anything.

Once constructed, a decoder is considered immutable, concurrency-safe, and can
decode into arbitrary outputs any amount of times. Also see `rd.Json` and
`rd.Form` for type-specific semantics.
*/
type Decoder interface{ Decode(interface{}) error }

/*
Represents an immutable string set, answering the question "is a particular key
contained in this set". All decoders in this package implement either
`rd.Haser` or `rd.Haserer`, answering the question "was the key present at the
top level of the request".
*/
type Haser interface{ Has(string) bool }

/*
Converts to `rd.Haser`. Implemented by all decoder types in this package. For
`rd.Form`, this is just a cast. For `rd.Json`, this involves reparsing the
JSON to build a set of keys.
*/
type Haserer interface{ Haser() Haser }

// Converts to `rd.Set`. Implemented by all decoder types in this package.
type Setter interface{ Set() Set }

/*
Interface for types that decode from `[]string`. Useful for parsing lists from
form-encoded sources such as URL queries and form bodies. Should be implemented
by any non-slice type that wants to be parsed from a list. Slice types don't
need this; this package handles them automatically, by parsing items
individually.
*/
type SliceParser interface{ ParseSlice([]string) error }

/*
Missing part of the "encoding" package. Commonly implemented by various types
across various libraries. If implemented, this is automatically used when
decoding strings from URL queries and form bodies. This is NOT used when
decoding from JSON; instead, types are expected to implement either
`json.Unmarshaler` or `encoding.TextUnmarshaler`.
*/
type Parser interface{ Parse(string) error }

// Shortcut for `rd.Decode` that panics on errors.
func TryDecode(req *http.Request, out interface{}) {
	try(Decode(req, out))
}

/*
Decodes an arbitrary request into an arbitrary Go structure. Transparently
supports multiple content types: URL query, URL-encoded body, multipart body,
and JSON. Read-only requests are decoded ONLY from the URL query, and other
requests are decoded ONLY from the body. For non-JSON requests, the output must
be a struct pointer. For JSON requests, the output may be a pointer to
anything.

If the request is multipart, this also populates `req.MultipartForm` as a side
effect. Downloaded files become available via `req.MultipartForm.File`.

Unlike `rd.Download`, this uses stream decoding for JSON, without buffering the
entire body in RAM. For other content types, this should perform identically to
`rd.Download`.
*/
func Decode(req *http.Request, out interface{}) error {
	if req == nil || out == nil {
		return nil
	}

	if isReqReadOnly(req) {
		return Form(req.URL.Query()).Decode(out)
	}

	typ := reqContentType(req)

	switch typ {
	case TypeJson:
		return errBadReq(json.NewDecoder(req.Body).Decode(out))

	case TypeForm:
		var dec Form
		err := dec.Download(req)
		if err != nil {
			return err
		}
		return dec.Decode(out)

	case TypeMulti:
		var dec Form
		err := dec.DownloadMultipart(req)
		if err != nil {
			return err
		}
		return dec.Decode(out)

	default:
		return errContentType(typ)
	}
}

// Shortcut for `rd.Download` that panics on errors.
func TryDownload(req *http.Request) Dec {
	dec, err := Download(req)
	try(err)
	return dec
}

/*
Downloads and partially decodes the request, returning a fully-buffered decoder,
appropriately chosen for the request type. Transparently supports multiple
content types: URL query, URL-encoded body, multipart body, and JSON. For
read-only requests, returns `rd.Form` populated ONLY from the URL query. For
JSON requests, returns `rd.Json` containing the fully-buffered response body,
without any decoding or modification. For URL-encoded requests and multipart
requests, returns `rd.Form` populated from the request body.

If the request is multipart, this also populates `req.MultipartForm` as a side
effect. Downloaded files become available via `req.MultipartForm.File`.
*/
func Download(req *http.Request) (Dec, error) {
	if req == nil {
		return nop{}, nil
	}

	if isReqReadOnly(req) {
		return Form(req.URL.Query()), nil
	}

	typ := reqContentType(req)

	switch typ {
	case TypeJson:
		var dec Json
		err := dec.Download(req)
		if err != nil {
			return nil, err
		}
		return dec, nil

	case TypeForm:
		var dec Form
		err := dec.DownloadForm(req)
		if err != nil {
			return nil, err
		}
		return dec, nil

	case TypeMulti:
		var dec Form
		err := dec.DownloadMultipart(req)
		if err != nil {
			return nil, err
		}
		return dec, nil

	default:
		return nil, errContentType(typ)
	}
}
