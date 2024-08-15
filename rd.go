package rd

import (
	"encoding/json"
	"net/http"
)

const (
	Type      = `Content-Type`
	TypeJson  = `application/json`
	TypeForm  = `application/x-www-form-urlencoded`
	TypeMulti = `multipart/form-data`

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
	err := Decode(req, out)
	if err != nil {
		panic(err)
	}
}

/*
Decodes an arbitrary request into an arbitrary Go structure. Uses the request's
`Content-Type` header to choose the decoding method.

When `Content-Type` is present but unrecognized, returns an error.

When `Content-Type` is missing and the request does have a body, returns an
error.

When `Content-Type` is missing and the request doesn't have a body, treats the
request's URL query exactly like the body of a formdata request. See below.

When `Content-Type` is `rd.TypeForm` (often called "formdata"), decodes the
request's body via `rd.Form`. The output must be a pointer to a struct. See
`rd.Form` about how this decoding works.

When `Content-Type` is `rd.TypeMulti`, decodes the request's text data via
`rd.Form`, and populates `req.MultipartForm` as a side effect. Downloaded files
become available via `req.MultipartForm.File`.

When `Content-Type` is `rd.TypeJson`, decodes the body into the output in a
streaming fashion, using `json.Decoder`. The output must be a pointer to any
value compatible with the structure of the provided JSON.
*/
func Decode(req *http.Request, out interface{}) error {
	if req == nil || out == nil {
		return nil
	}

	typ := reqContentType(req)

	switch typ {
	case ``:
		if reqHasBody(req) {
			return errContentType(typ)
		}
		return Form(reqQuery(req)).Decode(out)

	case TypeForm:
		var dec Form
		err := dec.DownloadForm(req)
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

	case TypeJson:
		body := req.Body
		if body == nil {
			return nil
		}
		return errBadReq(json.NewDecoder(body).Decode(out))

	default:
		return errContentType(typ)
	}
}

// Shortcut for `rd.Download` that panics on errors.
func TryDownload(req *http.Request) Dec {
	dec, err := Download(req)
	if err != nil {
		panic(err)
	}
	return dec
}

/*
Downloads the request's data, using the request's `Content-Type` header to
choose the appropriate decoder type. Unlike `Decode`, this always buffers
the request data in memory.

When `Content-Type` is present but unrecognized, returns an error.

When `Content-Type` is missing and the request does have a body, returns an
error.

When `Content-Type` is missing and the request doesn't have a body, returns
`rd.Form` with the request's URL query.

When `Content-Type` is `rd.TypeForm` (often called "formdata"), returns
`rd.Form` with the request's body.

When `Content-Type` is `rd.TypeMulti`, returns `rd.Form` with the text component
of the request body, and populates `req.MultipartForm` as a side effect.
Downloaded files become available via `req.MultipartForm.File`.

When `Content-Type` is `rd.TypeJson`, returns `rd.Json` containing the
downloaded response body, without any decoding or modification.
*/
func Download(req *http.Request) (Dec, error) {
	if req == nil {
		return decEmpty{}, nil
	}

	typ := reqContentType(req)

	switch typ {
	case ``:
		if reqHasBody(req) {
			return nil, errContentType(typ)
		}
		return Form(reqQuery(req)), nil

	case TypeForm:
		var dec Form
		err := dec.DownloadForm(req)
		return dec, err

	case TypeMulti:
		var dec Form
		err := dec.DownloadMultipart(req)
		return dec, err

	case TypeJson:
		var dec Json
		err := dec.Download(req)
		return dec, err

	default:
		return nil, errContentType(typ)
	}
}
