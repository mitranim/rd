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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
Request decoder. Decodes request inputs into structs. Transparently supports
multiple content types. Once constructed, a decoder is considered immutable,
concurrency-safe, and can be used any amount of times.
*/
type Reqdec struct {
	json map[string]json.RawMessage
	form url.Values
}

func FromJson(reader io.Reader) (Reqdec, error) {
	var out Reqdec
	err := readJsonOptional(reader, &out.json)
	return out, err
}

/*
Constructs a decoder that will decode from the given URL values. The values may
come from any source, such as request query, POST formdata, manually
constructed, etc. This should only be used in edge cases, such as when decoding
simultaneously from POST body and from URL query. For the general case,
construct your decoders using `Download`.
*/
func FromVals(vals url.Values) Reqdec {
	return Reqdec{form: vals}
}

/*
Shortcut for constructing a decoder via `FromVals` from the URL query of the
given request.
*/
func FromReqQuery(req *http.Request) Reqdec {
	return FromVals(req.URL.Query())
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
		return Reqdec{form: req.Form}, errWith(err, http.StatusBadRequest)
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
		return Reqdec{form: req.PostForm}, errWith(err, http.StatusBadRequest)

	case "multipart/form-data":
		// 32 MB, same as the default in the "http" package.
		// TODO make configurable.
		const maxmem = 32 << 20
		err := req.ParseMultipartForm(maxmem)
		out := Reqdec{form: url.Values(req.MultipartForm.Value)}
		if errors.Is(err, multipart.ErrMessageTooLarge) {
			return out, errWith(err, http.StatusRequestEntityTooLarge)
		}
		return out, errWith(err, http.StatusBadRequest)

	case "application/json":
		out, err := FromJson(req.Body)
		return out, errWith(err, http.StatusBadRequest)

	case "":
		return Reqdec{}, errWith(fmt.Errorf("unspecified request body type"), http.StatusBadRequest)

	default:
		return Reqdec{}, errWith(fmt.Errorf("unsupported request body type: %v", contentType), http.StatusBadRequest)
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
		return err
	}

	return refut.TraverseStructRtype(rootRval.Type(), func(sfield reflect.StructField, path []int) error {
		fieldName := sfieldJsonName(sfield)
		if fieldName == "" {
			return nil
		}

		if self.json != nil {
			return self.jsonDecodeAt(rootRval, fieldName, sfield.Type, path)
		}
		if self.form != nil {
			return self.formDecodeAt(rootRval, fieldName, sfield.Type, path)
		}
		return nil
	})
}

func (self Reqdec) jsonDecodeAt(rootRval reflect.Value, key string, rtype reflect.Type, path []int) error {
	val, has := self.json[key]
	if !has {
		return nil
	}

	if bytes.Equal(val, nullBytes) {
		zeroAt(rootRval, path)
		return nil
	}

	return json.Unmarshal(val, fieldPtrAt(rootRval, path))
}

func (self Reqdec) formDecodeAt(rootRval reflect.Value, key string, rtype reflect.Type, path []int) error {
	vals, has := self.form[key]
	if !has {
		return nil
	}

	if reflect.PtrTo(rtype).Implements(sliceParserRtype) {
		return fieldPtrAt(rootRval, path).(SliceParser).ParseSlice(vals)
	}

	if refut.RtypeDeref(rtype).Kind() == reflect.Slice {
		return untext.ParseSlice(vals, fieldPtrAt(rootRval, path))
	}

	val := self.form.Get(key)
	if val == "" {
		zeroAt(rootRval, path)
		return nil
	}

	return untext.Parse(self.form.Get(key), fieldPtrAt(rootRval, path))
}

/*
Returns true if the request body contains the specified key.
*/
func (self Reqdec) Has(key string) bool {
	_, ok := self.json[key]
	if ok {
		return ok
	}
	_, ok = self.form[key]
	return ok
}
