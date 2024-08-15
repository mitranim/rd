package rd

import (
	"fmt"
	"io"
	"net/http"
	r "reflect"
	"strconv"
)

/*
Wraps another error, adding an HTTP status code. Some errors returned by this
package are wrapped with codes such as 400 and 500.
*/
type Err struct {
	Status int   `json:"status"`
	Cause  error `json:"cause"`
}

// Implement a hidden interface in "errors".
func (self Err) Unwrap() error { return self.Cause }

// Returns `.Status`. Implements a hidden interface supported by
// `github.com/mitranim/rout`.
func (self Err) HttpStatusCode() int { return self.Status }

// Implement the `error` interface.
func (self Err) Error() string { return bytesString(self.AppendTo(nil)) }

// Appends the error representation. Used internally by `.Error`.
func (self Err) AppendTo(buf []byte) []byte {
	buf = append(buf, `[rd] error`...)

	if self.Status != 0 {
		buf = append(buf, ` (HTTP status `...)
		buf = strconv.AppendInt(buf, int64(self.Status), 10)
		buf = append(buf, `)`...)
	}

	cause := self.Cause
	if cause != nil {
		buf = append(buf, `: `...)
		impl, _ := cause.(interface{ AppendTo([]byte) []byte })
		if impl != nil {
			buf = impl.AppendTo(buf)
		} else {
			buf = append(buf, cause.Error()...)
		}
	}

	return buf
}

// Implement `fmt.Formatter`.
func (self Err) Format(out fmt.State, verb rune) {
	if out.Flag('+') {
		msg := self.AppendTo(nil)
		_, _ = out.Write(msg)

		if self.Cause != nil {
			if len(msg) > 0 {
				_, _ = io.WriteString(out, "\ncause:\n")
			}
			fmt.Fprintf(out, `%+v`, self.Cause)
		}
		return
	}

	if out.Flag('#') {
		type Error Err
		fmt.Fprintf(out, `%#v`, Error(self))
		return
	}

	_, _ = out.Write(self.AppendTo(nil))
}

func errBadReq(err error) error {
	if err == nil {
		return nil
	}
	return Err{http.StatusBadRequest, err}
}

func errInternal(err error) error {
	if err == nil {
		return nil
	}
	return Err{http.StatusInternalServerError, err}
}

func errInvalidPtr(val r.Value) error {
	return errInternal(fmt.Errorf(`expected settable struct pointer, got %v`, val))
}

func errParse(err error, input string, out r.Type) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(`failed to parse %q into %v: %v`, input, out, err)
}

func errContentType(typ string) error {
	if typ == `` {
		return errBadReq(fmt.Errorf(`missing content type`))
	}
	return errBadReq(fmt.Errorf(`unsupported content type %q`, typ))
}

var errJsonEof = errInternal(fmt.Errorf(`unexpected %w during JSON decoding`, io.EOF))

var errUnreachable = errInternal(fmt.Errorf(`unexpected violation of internal invariant`))
