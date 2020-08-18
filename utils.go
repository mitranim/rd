package reqdec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

/* Misc */

func isSlice(value interface{}) bool {
	if value == nil {
		return false
	}
	rtype := reflect.TypeOf(value)
	return refut.RtypeDeref(rtype).Kind() == reflect.Slice
}

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

// Allows the reader's content to be empty.
func readJsonOptional(reader io.Reader, out interface{}) error {
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return nil
	}

	return json.Unmarshal(content, out)
}
