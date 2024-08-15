package rd

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	r "reflect"
)

/*
Missing feature of the standard library: decodes URL-encoded and multipart
requests into Go structs. Implements `rd.Decoder`. Transparently used by
`rd.Decode` and `rd.Download` for appropriate request methods and content
types.

Similarities with "encoding/json":

	* Uses reflection to decode into arbitrary outputs.

	* Uses the "json" field tag.

	* Supports embedded structs.

	* Supports arbitrary field types.

	* Supports customizable decoding via `encoding.TextUnmarshaler` and
	  `rd.Parser` interfaces.

Differences from "encoding/json":

	* The top-level value must be a struct.

	* Doesn't support nested non-embedded structs.

	* Decodes only into fields with a "json" name, ignoring un-named fields.

	* For source fields which are "null", zeroes the corresponding fields of the
	  output struct, instead of leaving them as-is. "null" is defined as:

		* []string(nil)

		* []string{}

		* []string{``}

	* Has better performance.
*/
type Form url.Values

// Downloads the request body and populates the receiver based on the given
// content type. Used by `(*rd.Form).Download`.
func (self *Form) DownloadBody(req *http.Request, typ string) error {
	switch typ {
	case TypeForm:
		return self.DownloadForm(req)
	case TypeMulti:
		return self.DownloadMultipart(req)
	default:
		return errContentType(typ)
	}
}

// Assumes that the request has a URL-encoded body, downloads that body as a
// side effect, and populates the receiver.
func (self *Form) DownloadForm(req *http.Request) error {
	if req == nil {
		self.Zero()
		return nil
	}

	err := req.ParseForm()
	if err != nil {
		return errBadReq(err)
	}

	// Note: `(*Request).ParseForm` populates two fields:
	//   * `Request.Form`     -> from BOTH URL and body.
	//   * `Request.PostForm` -> from ONLY body.
	*self = Form(req.PostForm)
	return nil
}

/*
Assumes that the request has a multipart body, downloads that body as a side
effect, and populates the receiver. Uses the default buffer size of 32
megabytes.
*/
func (self *Form) DownloadMultipart(req *http.Request) error {
	return self.DownloadMultipartWith(req, BufSize)
}

/*
Assumes that the request has a multipart body, downloads that body as a side
effect, and populates the receiver. Passes the provided buffer size to
`(*http.Request).ParseMultipartForm`.
*/
func (self *Form) DownloadMultipartWith(req *http.Request, maxMem int64) error {
	if req == nil {
		self.Zero()
		return nil
	}

	err := req.ParseMultipartForm(maxMem)
	if err != nil {
		if errors.Is(err, multipart.ErrMessageTooLarge) {
			return Err{http.StatusRequestEntityTooLarge, err}
		}
		return errBadReq(err)
	}

	if req.MultipartForm == nil {
		self.Zero()
	} else {
		*self = Form(req.MultipartForm.Value)
	}
	return nil
}

// Deletes all key-values from the receiver.
func (self *Form) Zero() {
	if self == nil {
		return
	}
	for key := range *self {
		delete(*self, key)
	}
}

// Implements `rd.Parser` via `url.ParseQuery`.
func (self *Form) Parse(src string) error {
	val, err := url.ParseQuery(src)
	if err != nil {
		return err
	}
	*self = Form(val)
	return nil
}

// Implements `encoding.TextUnmarshaler`.
func (self *Form) UnmarshalText(src []byte) error {
	return self.Parse(string(src))
}

// Implement `rd.Haser`. Returns true if the key is present in the query map,
// regardless of its value.
func (self Form) Has(key string) bool {
	_, ok := self[key]
	return ok
}

// Implement `rd.Haserer` by returning self..
func (self Form) Haser() Haser { return self }

/*
Implement `rd.Setter` by creating an `rd.Set` composed of the keys present in
the form decoder.
*/
func (self Form) Set() Set {
	out := make(Set, len(self))
	for key := range self {
		out.Add(key)
	}
	return out
}

/*
Implement `rd.Decoder`, decoding into a struct. See `rd.Form` for the decoding
semantics.
*/
func (self Form) Decode(outVal interface{}) (err error) {
	if !(len(self) > 0) {
		return nil
	}

	defer trans(&err, errBadReq)

	out, err := derefStruct(r.ValueOf(outVal))
	if err != nil {
		return err
	}

	for _, field := range loadJsonFields(out.Type()) {
		err := self.decodeField(out, field)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self Form) decodeField(root r.Value, field jsonField) error {
	input, ok := self[field.Name]
	if !ok {
		return nil
	}

	if isSliceEmpty(input) {
		zeroAt(root, field.Path)
		return nil
	}

	out := derefAllocAt(root, field.Path)

	impl, _ := out.Addr().Interface().(SliceParser)
	if impl != nil {
		return impl.ParseSlice(input)
	}

	if out.Kind() == r.Slice {
		return parseSlice(input, out)
	}

	return Parse(input[0], out)
}

func reqQuery(req *http.Request) url.Values {
	if req == nil {
		return nil
	}
	url := req.URL
	if url == nil {
		return nil
	}
	if url.RawQuery == `` {
		return nil
	}
	return url.Query()
}
