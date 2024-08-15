package rd_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	r "reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mitranim/rd"
)

var (
	typeInt        = r.TypeOf((*int)(nil)).Elem()
	typeBool       = r.TypeOf((*bool)(nil)).Elem()
	typeString     = r.TypeOf((*string)(nil)).Elem()
	typeBytes      = r.TypeOf((*[]byte)(nil)).Elem()
	typeTime       = r.TypeOf((*time.Time)(nil)).Elem()
	typeTimeParser = r.TypeOf((*TimeParser)(nil)).Elem()
)

var numTypes = []r.Type{
	r.TypeOf((*uint8)(nil)).Elem(),
	r.TypeOf((*uint16)(nil)).Elem(),
	r.TypeOf((*uint32)(nil)).Elem(),
	r.TypeOf((*uint64)(nil)).Elem(),
	r.TypeOf((*uint)(nil)).Elem(),
	r.TypeOf((*int8)(nil)).Elem(),
	r.TypeOf((*int16)(nil)).Elem(),
	r.TypeOf((*int32)(nil)).Elem(),
	r.TypeOf((*int64)(nil)).Elem(),
	r.TypeOf((*int)(nil)).Elem(),
	r.TypeOf((*float32)(nil)).Elem(),
	r.TypeOf((*float64)(nil)).Elem(),
}

type Outer struct {
	Embed
	Inner    Inner  `json:"inner"    db:"inner"`
	OuterStr string `json:"outerStr" db:"outer_str"`
}

type PtrOuter struct {
	*Embed
	Inner    Inner  `json:"inner"    db:"inner"`
	OuterStr string `json:"outerStr" db:"outer_str"`
}

type Inner struct {
	InnerStr string `json:"innerStr" db:"inner_str"`
	InnerNum int    `json:"innerNum" db:"inner_num"`
}

type Embed struct {
	EmbedStr string `json:"embedStr" db:"embed_str"`
	EmbedNum int    `json:"embedNum" db:"embed_num"`
}

var testOuter = Outer{
	Embed:    Embed{EmbedStr: `embed val`, EmbedNum: 10},
	Inner:    Inner{InnerStr: `inner val`, InnerNum: 20},
	OuterStr: `outer val`,
}

var testOuterSimple = Outer{
	Embed:    Embed{EmbedStr: `embed val`, EmbedNum: 10},
	OuterStr: `outer val`,
}

var testPtrOuterSimple = PtrOuter{
	Embed:    &Embed{EmbedStr: `embed val`, EmbedNum: 10},
	OuterStr: `outer val`,
}

var testOuterQuery = url.Values{
	`embedStr`: {`embed val`},
	`embedNum`: {`10`},
	`outerStr`: {`outer val`},
}

var testOuterQuerySet = rd.Set{
	`embedStr`: struct{}{},
	`embedNum`: struct{}{},
	`outerStr`: struct{}{},
}

const testOuterJson = `{
	"embedStr": "embed val",
	"embedNum": 10,
	"inner": {
		"innerStr": "inner val",
		"innerNum": 20
	},
	"outerStr": "outer val"
}`

const testOuterSimpleJson = `{
	"embedStr": "embed val",
	"embedNum": 10,
	"outerStr": "outer val"
}`

var testOuterJsonSet = rd.Set{
	`embedStr`: struct{}{},
	`embedNum`: struct{}{},
	`inner`:    struct{}{},
	`outerStr`: struct{}{},
}

// nolint:structcheck,unused,govet
type TarUnusable struct {
	Untagged0 string
	Untagged1 string `json:"-"`
	private   string `json:"private"`
	_         string `json:"blank"`
}

var unusableVals = url.Values{
	`Untagged0`: {`untagged0 val`},
	`Untagged1`: {`untagged1 val`},
	`private`:   {`private val`},
	`blank`:     {`blank val`},
}

var testUrlQuery = url.Values{
	`one`:  {``},
	`two`:  {`three`},
	`four`: {`five`, `six`},
}

var testBodyQuery = url.Values{
	`seven`: {``},
	`eight`: {`nine`},
	`ten`:   {`eleven`, `twelve`},
}

var testJsonStr = `{
	"one": 10,
	"two": true,
	"three": ["four"]
}`

type TarVoid struct{}

type TarPair struct {
	One []int `json:"one"`
	Two []int `json:"two"`
}

type TarInt struct {
	Val int `json:"val"`
}

type TarPtrInt struct {
	Val *int `json:"val"`
}

type TarSliceInt struct {
	Val []int `json:"val"`
}

type TarSlicePtrInt struct {
	Val []*int `json:"val"`
}

func ptrInt(val int) *int { return &val }

var testNums = []int{0, 1, 2, 3, 4, 8, 16, 32}

var nopQueries = []url.Values{
	nil,
	{`nop`: nil},
	{`nop`: {}},
	{`nop`: {``}},
	{`nop`: {`any`}},
}

var zeroQueries = []url.Values{
	{`val`: nil},
	{`val`: {}},
	{`val`: {``}},
}

var numQueries = []url.Values{
	{`val`: {`20`}},
	{`val`: {`20`, `30`}},
	{`val`: {`20`, `30`, `40`}},
}

var numOutputs = [][]int{
	{20},
	{20, 30},
	{20, 30, 40},
}

var numPtrOutputs = [][]*int{
	{ptrInt(20)},
	{ptrInt(20), ptrInt(30)},
	{ptrInt(20), ptrInt(30), ptrInt(40)},
}

func eq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if !r.DeepEqual(exp, act) {
		t.Fatalf(`
expected (detailed):
	%#[1]v
actual (detailed):
	%#[2]v
expected (simple):
	%[1]v
actual (simple):
	%[2]v
`, exp, act)
	}
}

func errs(t testing.TB, msg string, err error) {
	if err == nil {
		t.Fatalf(`expected an error with %q, got none`, msg)
	}

	str := err.Error()
	if !strings.Contains(str, msg) {
		t.Fatalf(`expected an error with a message containing %q, got %q`, msg, str)
	}
}

func try(err error) {
	if err != nil {
		panic(err)
	}
}

type Req http.Request

func (self Req) Init() Req {
	if self.Method == `` {
		self.Method = http.MethodGet
	}
	if self.URL == nil {
		self.URL = new(url.URL)
	}
	if self.Header == nil {
		self.Header = http.Header{}
	}
	return self
}

func (self Req) Ptr() *http.Request {
	self = self.Init()
	return (*http.Request)(&self)
}

func (self Req) Post() Req {
	self.Method = http.MethodPost
	return self
}

func (self Req) TypeJson() Req { return self.Type(rd.TypeJson) }

func (self Req) TypeForm() Req { return self.Type(rd.TypeForm) }

func (self Req) Query(val url.Values) Req {
	self = self.Init()
	self.URL.RawQuery = val.Encode()
	return self
}

func (self Req) BodyReadCloser(val io.ReadCloser) Req {
	self.Body = val
	if val == nil {
		self.ContentLength = 0
	} else {
		self.ContentLength = -1
	}
	return self
}

func (self Req) BodyReader(val io.Reader) Req {
	if val == nil {
		return self.BodyReadCloser(nil)
	}
	return self.BodyReadCloser(io.NopCloser(val))
}

func (self Req) BodyString(val string) Req {
	if val == `` {
		return self.BodyReader(nil)
	}
	return self.BodyReader(strings.NewReader(val))
}

func (self Req) BodyJson(val string) Req {
	return self.TypeJson().BodyString(val)
}

func (self Req) BodyForm(val url.Values) Req {
	return self.TypeForm().BodyString(val.Encode())
}

func (self Req) BodyMulti(src url.Values) Req {
	typ, reader := queryToMultipart(src)
	return self.Type(typ).BodyReader(reader)
}

func (self Req) Type(val string) Req {
	self = self.Init()
	self.Header.Set(rd.Type, val)
	return self
}

func queryToMultipart(src url.Values) (string, io.Reader) {
	var buf bytes.Buffer
	wri := multipart.NewWriter(&buf)

	for key, vals := range src {
		for _, val := range vals {
			try(wri.WriteField(key, val))
		}
	}
	try(wri.Close())

	return wri.FormDataContentType(), &buf
}

func parseNew(src string, typ r.Type) r.Value {
	out := r.New(typ).Elem()
	try(rd.Parse(src, out))
	return out
}

type TimeParser time.Time

func (self *TimeParser) Parse(src string) error {
	val, err := time.Parse(time.RFC3339, src)
	if err != nil {
		return err
	}
	*self = TimeParser(val)
	return nil
}

func iter(count int) []struct{} { return make([]struct{}, count) }

func set(vals ...string) rd.Set {
	if !(len(vals) > 0) {
		return nil
	}

	out := make(rd.Set, len(vals))
	for _, val := range vals {
		out.Add(val)
	}
	return out
}

// Exists for verifying that `SliceParser` is invoked for non-slices, not just
// for slices.
type SliceParserStruct struct{ Inner []int }

func (self *SliceParserStruct) ParseSlice(vals []string) error {
	out := self.Inner[:0]

	for _, val := range vals {
		num, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		out = append(out, num)
	}

	self.Inner = out
	return nil
}
