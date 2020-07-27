package reqdec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
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
	return derefRtype(rtype).Kind() == reflect.Slice
}

func derefRtype(rtype reflect.Type) reflect.Type {
	for rtype != nil && rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	return rtype
}

/*
Recursively dereferences a `reflect.Value` until it's not a pointer type. Panics
if any pointer in the sequence is nil.
*/
func derefRval(rval reflect.Value) reflect.Value {
	for rval.Kind() == reflect.Ptr {
		rval = rval.Elem()
	}
	return rval
}

/*
Derefs the provided value until it's no longer a pointer, allocating as
necessary. Returns a non-pointer value. The input value must be settable or a
non-nil pointer, otherwise this causes a panic.
*/
func derefAllocRval(rval reflect.Value) reflect.Value {
	for rval.Kind() == reflect.Ptr {
		if rval.IsNil() {
			rval.Set(reflect.New(rval.Type().Elem()))
		}
		rval = rval.Elem()
	}
	return rval
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

/*
TODO: consider passing the entire path from the root value rather than the field
index. This is more expensive but allows the caller to choose to allocate deeply
nested fields on demand.
*/
func traverseStructRvalueFields(rval reflect.Value, fun func(reflect.Value, int) error) error {
	rval = derefRval(rval)
	rtype := rval.Type()
	if rtype.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct, got a %q", rtype)
	}

	for i := 0; i < rtype.NumField(); i++ {
		sfield := rtype.Field(i)
		if !isStructFieldPublic(sfield) {
			continue
		}

		/**
		If this is an embedded struct, traverse its fields as if they're in the
		parent struct.
		*/
		if sfield.Anonymous && derefRtype(sfield.Type).Kind() == reflect.Struct {
			err := traverseStructRvalueFields(rval.Field(i), fun)
			if err != nil {
				return err
			}
			continue
		}

		err := fun(rval, i)
		if err != nil {
			return err
		}
	}

	return nil
}

func isStructFieldPublic(sfield reflect.StructField) bool {
	return sfield.PkgPath == ""
}

func structFieldName(sfield reflect.StructField) string {
	return jsonTagToFieldName(sfield.Tag.Get("json"))
}

func jsonTagToFieldName(tag string) string {
	if tag == "-" {
		return ""
	}
	index := strings.IndexRune(tag, ',')
	if index >= 0 {
		return tag[:index]
	}
	return tag
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
