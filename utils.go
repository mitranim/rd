package reqdec

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/mitranim/refut"
)

/* Errors */

// Describes an error with an optional HTTP status code.
type Err struct {
	HttpStatus int
	Cause      error
}

// Implement `error`.
func (self Err) Error() string {
	if self == (Err{}) {
		return ``
	}
	msg := `reqdec error`
	if self.HttpStatus != 0 {
		msg += fmt.Sprintf(` %v`, self.HttpStatus)
	}
	if self.Cause != nil {
		msg += `: ` + self.Cause.Error()
	}
	return msg
}

// Implement a hidden interface in "errors".
func (self Err) Is(other error) bool {
	if self.Cause != nil && errors.Is(self.Cause, other) {
		return true
	}
	err, ok := other.(Err)
	return ok && err.HttpStatus == self.HttpStatus
}

// Implement a hidden interface in "errors".
func (self Err) Unwrap() error {
	return self.Cause
}

func errWith(err error, status int) error {
	if err == nil {
		return nil
	}

	var local Err
	if errors.As(err, &local) {
		local.HttpStatus = status
		return local
	}

	return Err{HttpStatus: status, Cause: err}
}

/* Misc */

func settableRval(input interface{}) (reflect.Value, error) {
	rval := reflect.ValueOf(input)
	rtype := rval.Type()
	if rtype.Kind() != reflect.Ptr {
		return rval, fmt.Errorf(`expected a pointer, got a %q`, rtype)
	}
	rval = rval.Elem()
	if !rval.CanSet() {
		return rval, fmt.Errorf(`can't set into non-settable value of type %q`, rtype)
	}
	return rval, nil
}

func settableStructRval(out interface{}) (reflect.Value, error) {
	rval, err := settableRval(out)
	if err != nil {
		return rval, err
	}
	if rval.Type().Kind() != reflect.Struct {
		return rval, fmt.Errorf("expected a struct pointer, got a %q", rval.Type())
	}
	return rval, nil
}

func sfieldJsonName(sfield reflect.StructField) string {
	return refut.TagIdent(sfield.Tag.Get("json"))
}

func isHttpMethodReadOnly(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

func readJsonOptional(src io.Reader, out interface{}) error {
	if src == nil {
		return nil
	}

	// Skips decoding when the input is empty.
	reader := bufio.NewReader(src)
	_, err := reader.Peek(1)
	if errors.Is(err, io.EOF) {
		return nil
	}

	return json.NewDecoder(reader).Decode(out)
}

var nullBytes = []byte(`null`)

func zeroAt(rval reflect.Value, path []int) {
	for _, ind := range path {
		for rval.Kind() == reflect.Ptr {
			if rval.IsNil() {
				return
			}
			rval = rval.Elem()
		}
		rval = rval.Field(ind)
	}

	if rval.IsValid() {
		rval.Set(reflect.Zero(rval.Type()))
	}
}

func fieldPtrAt(root reflect.Value, path []int) interface{} {
	return refut.RvalFieldByPathAlloc(root, path).Addr().Interface()
}

var sliceParserRtype = reflect.TypeOf((*SliceParser)(nil)).Elem()
