package rd

import (
	"encoding"
	"fmt"
	r "reflect"
	"strconv"
)

/*
Missing feature of the standard library: parse arbitrary strings into arbitrary
Go values. Used internally by `rd.Form.Decode`. Exported for enterprising
users. Adapted from "github.com/mitranim/untext". The output must be a settable
non-pointer. Its original value is ignored/overwritten. If the output
implements `rd.SliceParser`, the corresponding method is invoked automatically.
Otherwise it must be a slice of some concrete type, where each element is
parsed via `rd.Parse`. Unlike "encoding/json", this doesn't support parsing
into dynamically-typed `interface{}` values.
*/
func ParseSlice(inputs []string, out r.Value) error {
	impl, _ := out.Addr().Interface().(SliceParser)
	if impl != nil {
		return impl.ParseSlice(inputs)
	}
	return parseSlice(inputs, out)
}

func parseSlice(inputs []string, out r.Value) error {
	if inputs == nil {
		out.Set(r.Zero(out.Type()))
		return nil
	}

	buf := r.MakeSlice(out.Type(), len(inputs), len(inputs))

	for i, input := range inputs {
		err := Parse(input, derefAlloc(buf.Index(i)))
		if err != nil {
			return err
		}
	}

	// TODO: does this make an additional copy?
	out.Set(buf)
	return nil
}

/*
Missing feature of the standard library: parse arbitrary text into arbitrary Go
value. Used internally by `rd.Form.Decode`. Exported for enterprising users.
Adapted from "github.com/mitranim/untext". The output must be a settable
non-pointer. Its original value is ignored/overwritten. If the output
implements `rd.Parser` or `encoding.TextUnmarshaler`, the corresponding method
is invoked automatically. Otherwise the output must be a "well-known" Go type:
number, bool, string, or byte slice. Unlike "encoding/json", this doesn't
support parsing into dynamically-typed `interface{}` values.
*/
func Parse(input string, out r.Value) error {
	ptr := out.Addr().Interface()

	parser, _ := ptr.(Parser)
	if parser != nil {
		return parser.Parse(input)
	}

	unmarshaler, _ := ptr.(encoding.TextUnmarshaler)
	if unmarshaler != nil {
		return unmarshaler.UnmarshalText(stringToBytesUnsafe(input))
	}

	typ := out.Type()
	kind := typ.Kind()

	switch kind {
	case r.Int8, r.Int16, r.Int32, r.Int64, r.Int:
		val, err := strconv.ParseInt(input, 10, typeBits(typ))
		out.SetInt(val)
		return errParse(err, input, typ)

	case r.Uint8, r.Uint16, r.Uint32, r.Uint64, r.Uint:
		val, err := strconv.ParseUint(input, 10, typeBits(typ))
		out.SetUint(val)
		return errParse(err, input, typ)

	case r.Float32, r.Float64:
		val, err := strconv.ParseFloat(input, typeBits(typ))
		out.SetFloat(val)
		return errParse(err, input, typ)

	case r.Bool:
		return parseBool(input, out)

	case r.String:
		out.SetString(input)
		return nil

	default:
		if typ.ConvertibleTo(typeBytes) {
			// Unavoidable copy?
			out.SetBytes([]byte(input))
			return nil
		}

		return fmt.Errorf(`failed to parse %q into %v: unsupported kind %v`, input, typ, kind)
	}
}

// Note: `strconv.ParseBool` is too permissive for our taste.
func parseBool(input string, out r.Value) error {
	switch input {
	case `true`:
		out.SetBool(true)
		return nil

	case `false`:
		out.SetBool(false)
		return nil

	default:
		return fmt.Errorf(`failed to parse %q into bool`, input)
	}
}
