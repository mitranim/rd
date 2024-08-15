package rd

import (
	"mime"
	"mime/multipart"
	"net/http"
	r "reflect"
	"strings"
	"sync"
	"unsafe"
)

var (
	typeBytes = r.TypeOf((*[]byte)(nil)).Elem()
)

/*
We should attempt to decode or download request body only when it's actually
present. When the body is not present, we should not require the client to
specify content type, as it's not needed. This clause uses `== 0`, rather than
`<= 0`, because the documentation for `http.Request` states that -1 means
unknown length.
*/
func reqHasBody(req *http.Request) bool {
	return req != nil && req.Body != nil && req.ContentLength != 0
}

func reqContentType(req *http.Request) string {
	val, _, _ := mime.ParseMediaType(req.Header.Get(Type))
	return val
}

/*
Allocation-free conversion. Reinterprets a byte slice as a string. Borrowed from
the standard library. Reasonably safe.
*/
func bytesString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

/*
Allocation-free conversion. Returns a byte slice backed by the provided string.
Mutations are reflected in the source string, unless it's backed by constant
storage, in which case they trigger a segfault. Reslicing is ok. Should be safe
as long as the resulting bytes are not mutated.
*/
func stringToBytesUnsafe(val string) []byte {
	type sliceHeader struct {
		_   uintptr
		len int
		cap int
	}
	slice := *(*sliceHeader)(unsafe.Pointer(&val))
	slice.cap = slice.len
	return *(*[]byte)(unsafe.Pointer(&slice))
}

type decEmpty struct{}

func (decEmpty) Download(*http.Request) error         { return nil }
func (decEmpty) Decode(interface{}) error             { return nil }
func (decEmpty) Files(string) []*multipart.FileHeader { return nil }
func (decEmpty) Has(string) bool                      { return false }
func (self decEmpty) Haser() Haser                    { return self }
func (decEmpty) Set() Set                             { return nil }

func trans(err *error, fun func(error) error) {
	if *err != nil {
		*err = fun(*err)
	}
}

func derefType(typ r.Type) r.Type {
	for typ != nil && typ.Kind() == r.Ptr {
		typ = typ.Elem()
	}
	return typ
}

func derefStruct(src r.Value) (r.Value, error) {
	val := src

	for val.Kind() == r.Ptr {
		if val.IsNil() {
			return val, errInvalidPtr(src)
		}
		val = val.Elem()
	}

	if val.Kind() != r.Struct || !val.CanSet() {
		return val, errInvalidPtr(src)
	}
	return val, nil
}

func derefAlloc(val r.Value) r.Value {
	for val.Kind() == r.Ptr {
		if val.IsNil() {
			val.Set(r.New(val.Type().Elem()))
		}
		val = val.Elem()
	}
	return val
}

func derefAllocAt(val r.Value, path []int) r.Value {
	val = derefAlloc(val)

	for len(path) > 0 {
		val = derefAlloc(val.Field(path[0]))
		path = path[1:]
	}

	return val
}

func zeroAt(val r.Value, path []int) {
	for _, index := range path {
		for val.Kind() == r.Ptr {
			if val.IsNil() {
				return
			}
			val = val.Elem()
		}
		val = val.Field(index)
	}

	if val.CanSet() {
		val.Set(r.Zero(val.Type()))
	}
}

func iter(count int) []struct{} { return make([]struct{}, count) }

func tagIdent(tag string) string {
	index := strings.IndexRune(tag, ',')
	if index >= 0 {
		tag = tag[:index]
	}
	if tag == `-` {
		return ``
	}
	return tag
}

func jsonName(field r.StructField) string {
	return tagIdent(field.Tag.Get(`json`))
}

func resliceInts(val *[]int, length int) { *val = (*val)[:length] }

func copyInts(src []int) []int {
	if src == nil {
		return nil
	}
	out := make([]int, len(src))
	copy(out, src)
	return out
}

func isPublic(pkgPath string) bool { return pkgPath == `` }

type jsonField struct {
	Name string
	Path []int
}

var jsonFieldCache sync.Map

// Susceptible to "thundering herd" but much better than no caching.
func loadJsonFields(typ r.Type) []jsonField {
	if typ == nil {
		return nil
	}

	val, ok := jsonFieldCache.Load(typ)
	if ok {
		return val.([]jsonField)
	}

	out := jsonFields(typ)
	jsonFieldCache.Store(typ, out)
	return out
}

func jsonFields(typ r.Type) (out []jsonField) {
	path := make([]int, 0, 8)
	for i := range iter(typ.NumField()) {
		appendJsonFields(&out, &path, typ, i)
	}
	return
}

func appendJsonFields(buf *[]jsonField, path *[]int, typ r.Type, index int) {
	defer resliceInts(path, len(*path))
	*path = append(*path, index)

	field := typ.Field(index)
	if !isPublic(field.PkgPath) {
		return
	}

	name := jsonName(field)
	if name != `` {
		*buf = append(*buf, jsonField{name, copyInts(*path)})
		return
	}

	if field.Anonymous {
		typ := derefType(field.Type)
		if typ.Kind() == r.Struct {
			for i := range iter(typ.NumField()) {
				appendJsonFields(buf, path, typ, i)
			}
		}
	}
}

func isSliceEmpty(val []string) bool {
	return !(len(val) > 0) || (len(val) == 1 && val[0] == ``)
}

func typeBits(typ r.Type) int {
	return int(typ.Size() * 8)
}
