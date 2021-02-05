/*
Tool for decoding an HTTP request into a struct. Transparently supports
different content types / encoding formats (JSON, URL-encoded form, multipart
form). Transparently supports different request methods; for read-only methods
such as GET parses inputs from the URL; for non-read-only methods parses inputs
ONLY from the body.

For URL-encoded and multipart forms, uses "github.com/mitranim/untext" to decode
strings into Go values. For JSON, uses "encoding/json".

Usage

Example:

	dec, err := reqdec.Download(req)
	if err != nil {} // ...

	var input struct {
	  FieldOne string `json:"field_one"`
	  FieldTwo int64  `json:"field_two"`
	}
	err = dec.DecodeStruct(&input)
	if err != nil {} // ...
*/
package reqdec

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"

	"github.com/mitranim/refut"
	"github.com/mitranim/untext"
)

/*
Interface for types that decode from `[]string`. Useful for parsing lists from
form-encoded sources such as URL queries and form bodies. This should be
implemented by any type that wants to be parsed from a list, but by itself is
not a slice. Slice types don't need this; they're automatically allocated and
each item is parsed individually.
*/
type SliceParser interface {
	ParseSlice([]string) error
}

/*
Request decoder obtained by calling `Download()`.
*/
type Reqdec struct {
	jsonDict map[string]json.RawMessage
	formDict url.Values
}

/*
Constructs a decoder that will decode from the given URL values. The values may
come from any source, such as request query, POST formdata, manually
constructed, etc. This should only be used in edge cases, such as when decoding
simulataneously from POST body and from URL query. For the general case,
construct your decoders using `Download`.
*/
func FromQuery(vals url.Values) Reqdec {
	return Reqdec{formDict: vals}
}

/*
Shortcut for constructing a decoder via `FromQuery` from the URL query of the
given request.
*/
func FromReqQuery(req *http.Request) Reqdec {
	return FromQuery(req.URL.Query())
}

/*
Downloads the HTTP request and returns a `Reqdec` populated with the request's
body, decoded in accordance with the declared content type.
*/
func Download(req *http.Request) (Reqdec, error) {
	if isHttpMethodReadOnly(req.Method) {
		/**
		Note: despite its name, `Request.ParseForm()` also parses values from
		the URL, populating them into `Request.Form`.
		*/
		err := req.ParseForm()
		if err != nil {
			return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: err}
		}
		return Reqdec{formDict: req.Form}, nil
	}

	contentType, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))

	switch contentType {
	case "application/x-www-form-urlencoded":
		/**
		Note: `Request.ParseForm()` parses both URL and request body, populating
		`Request.Form` with values from both and `Request.PostForm` with values
		only from the body. For non-read-only HTTP methods, such as POST, we
		explicitly use `Request.PostForm` to avoid accidentally taking inputs
		from the URL.
		*/
		err := req.ParseForm()
		if err != nil {
			return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: err}
		}
		return Reqdec{formDict: req.PostForm}, nil

	case "multipart/form-data":
		// 32 MB, same as the default in the "http" package.
		// TODO make configurable.
		const maxmem = 32 << 20
		err := req.ParseMultipartForm(maxmem)
		if err != nil {
			if errors.Is(err, multipart.ErrMessageTooLarge) {
				return Reqdec{}, Err{HttpStatus: http.StatusRequestEntityTooLarge, Cause: err}
			}
			return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: err}
		}
		return Reqdec{formDict: url.Values(req.MultipartForm.Value)}, nil

	case "application/json":
		var out Reqdec
		err := readJsonOptional(req.Body, &out.jsonDict)
		if err != nil {
			return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: err}
		}
		return out, nil

	case "":
		return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: fmt.Errorf("unspecified request body type")}

	default:
		return Reqdec{}, Err{HttpStatus: http.StatusBadRequest, Cause: fmt.Errorf("unsupported request body type: %v", contentType)}
	}
}

/*
Decodes the input into a struct. Conceptually similar to "json.Unmarshal" but
also works for URL-encoded and multipart forms, using
"github.com/mitranim/untext" to unmarshal text into Go values. Unmarshable
struct fields must be tagged with `json:"<fieldName>`.

Does not allocate inner pointer structs if none of their fields were found in
the input. This allows us to easily check if an inner struct has any inputs by
comparing it to nil.

TODO: support structs embedded as pointers rather than concrete. This should
allocate the embedded struct only after finding a value worth decoding into one
of its fields.
*/
func (self Reqdec) DecodeStruct(dest interface{}) error {
	rootRval, err := settableStructRval(dest)
	if err != nil {
		return Err{Cause: err}
	}

	return refut.TraverseStructRtype(rootRval.Type(), func(sfield reflect.StructField, path []int) error {
		fieldName := sfieldJsonName(sfield)

		if !self.Has(fieldName) {
			return nil
		}

		// If this is a nested struct, we allocate it only after finding
		// something worth decoding.
		fieldPtr := refut.RvalFieldByPathAlloc(rootRval, path).Addr().Interface()
		return self.DecodeAt(fieldName, fieldPtr)
	})
}

/*
Decodes the value at the given key into an arbitrary destination pointer.
*/
func (self Reqdec) DecodeAt(key string, dest interface{}) error {
	if self.jsonDict != nil {
		err := json.Unmarshal(self.jsonDict[key], dest)
		if err != nil {
			return Err{Cause: err}
		}
		return nil
	}

	vals := self.formDict[key]

	parser, ok := dest.(SliceParser)
	if ok {
		return parser.ParseSlice(vals)
	}

	/**
	Support unmarshaling a slice from `url.Values` where each value is included
	individually. Example:

		struct { Value []int64 `json:"value"` }

		"?value=10&value=20&value=30"

	*/
	if len(vals) > 0 && isSlice(dest) {
		return untext.UnmarshalSlice(vals, dest)
	}

	return untext.UnmarshalString(self.formDict.Get(key), dest)
}

/*
Returns true if the request body contains the specified key.
*/
func (self Reqdec) Has(key string) bool {
	_, ok := self.jsonDict[key]
	if ok {
		return ok
	}
	_, ok = self.formDict[key]
	return ok
}
