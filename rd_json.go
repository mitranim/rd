package rd

import (
	"encoding/json"
	"io"
	"net/http"
)

/*
Implements `rd.Decoder` via `json.Unmarshal`. Unlike other decoders, supports
arbitrary output types, not just structs. Unlike other decoders, this doesn't
directly implement `rd.Haser`, but it does implement `rd.Haserer` which has a
minor cost; see `rd.Json.Haser`.
*/
type Json []byte

/*
Fully downloads the request body and stores it as-is, without any modification
or validation. Used by `rd.Download`.
*/
func (self *Json) Download(req *http.Request) error {
	if req == nil || req.Body == nil {
		self.Zero()
		return nil
	}

	out, err := io.ReadAll(req.Body)
	if err != nil {
		return errBadReq(err)
	}

	*self = out
	return nil
}

/*
Implement `json.Unmarshaler` by storing the input as-is, exactly like
`json.RawMessage`. This allows to include `rd.Json` into other data structures,
deferring the decoding until later.
*/
func (self *Json) UnmarshalJSON(src []byte) error {
	if src == nil {
		*self = nil
	} else {
		out := make([]byte, len(src))
		copy(out, src)
		*self = out
	}
	return nil
}

// Clears the slice, preserving the capacity if any.
func (self *Json) Zero() {
	if self != nil && len(*self) > 0 {
		*self = (*self)[:0]
	}
}

/*
Implement `rd.Decoder` by calling `json.Unmarshal`. The output must be a non-nil
pointer to an arbitrary Go value.
*/
func (self Json) Decode(out interface{}) error {
	return errBadReq(json.Unmarshal(self, out))
}

// Implement `rd.Haserer` by calling `rd.Json.Set`.
func (self Json) Haser() Haser { return self.Set() }

/*
Implement `rd.Setter`. Returns an instance of `rd.Set` with the keys of the
top-level object in the JSON text. Assumes that JSON is either valid or
completely empty (only whitespace). Panics on malformed JSON.

Unlike other decoders provided by this package, `rd.Json.Haser` is not a free
cast; it has to re-parse the JSON to build the set of top-level object keys. It
uses a custom JSON decoder optimized for this particular operation, which
avoids reflection and performs much better than "encoding/json". The overhead
should be barely measurable.

Caution: for efficiency, this assumes that `rd.Json` is immutable, and performs
an unsafe cast from `[]byte` to `string`. Parts of the resulting string are
used as map keys. Mutating the JSON slice after calling this method will result
in undefined behavior. Mutating the resulting set is perfectly safe.
*/
func (self Json) Set() Set { return parseSet(bytesString(self)) }

/*
Simple string set backed by a Go map. Implements `rd.Haser`. Generated by
`rd.Json.Haser`.
*/
type Set map[string]struct{}

// Implements `rd.Haserer` by returning self.
func (self Set) Haser() Haser { return self }

// Implement `rd.Haser`. Returns true if the value is among the map's keys.
func (self Set) Has(val string) bool {
	_, ok := self[val]
	return ok
}

// Adds the value to the set.
func (self Set) Add(val string) { self[val] = struct{}{} }

// Deletes the value from the set.
func (self Set) Del(val string) { delete(self, val) }
