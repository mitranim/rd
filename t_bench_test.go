package rd_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/mitranim/rd"
)

func BenchmarkJson_Decode(b *testing.B) {
	dec := rd.Json(testOuterSimpleJson)
	var tar Outer
	b.ResetTimer()

	for range iter(b.N) {
		try(dec.Decode(&tar))
	}
}

func BenchmarkQuery_Decode(b *testing.B) {
	dec := rd.Form(testOuterQuery)
	var tar Outer
	b.ResetTimer()

	for range iter(b.N) {
		try(dec.Decode(&tar))
	}
}

func BenchmarkQuery_Parse_Decode(b *testing.B) {
	src := testOuterQuery.Encode()
	var tar Outer
	b.ResetTimer()

	for range iter(b.N) {
		vals, err := url.ParseQuery(src)
		try(err)
		try(rd.Form(vals).Decode(&tar))
	}
}

func BenchmarkSet_construct(b *testing.B) {
	for range iter(b.N) {
		haserNop(set(testSetKeys...))
	}
}

var haserNop = func(rd.Haser) {}

var testSetKeys = []string{`embedStr`, `embedNum`, `outerStr`}

func BenchmarkJson_Haser(b *testing.B) {
	dec := rd.Json(testOuterSimpleJson)
	test_Json_Haser(b, dec)
	b.ResetTimer()

	for range iter(b.N) {
		dec.Haser()
	}
}

func test_Json_Haser(t testing.TB, src rd.Json) {
	eq(
		t,
		rd.Set{
			`embedStr`: struct{}{},
			`embedNum`: struct{}{},
			`outerStr`: struct{}{},
		},
		src.Haser(),
	)
}

func Benchmark_parseSetWithStdlib(b *testing.B) {
	src := []byte(testOuterSimpleJson)
	test_parseSetWithStdlib(b, src)
	b.ResetTimer()

	for range iter(b.N) {
		parseSetWithStdlib(src)
	}
}

func test_parseSetWithStdlib(t testing.TB, src []byte) {
	eq(
		t,
		map[string]void{
			`embedStr`: struct{}{},
			`embedNum`: struct{}{},
			`outerStr`: struct{}{},
		},
		parseSetWithStdlib(src),
	)
}

func parseSetWithStdlib(src []byte) (out map[string]void) {
	try(json.Unmarshal(src, &out))
	return
}

type void struct{}

func (*void) UnmarshalJSON([]byte) error { return nil }

func BenchmarkJson_Haser_empty(b *testing.B) {
	for range iter(b.N) {
		haserNop(rd.Json{}.Haser())
	}
}

func Benchmark_json_parse_mixed_stdlib(b *testing.B) {
	src := []byte(jsonSrcMixed)
	b.ResetTimer()

	for range iter(b.N) {
		parseSetWithStdlib(src)
	}
}

func Benchmark_json_parse_mixed_ours(b *testing.B) {
	dec := rd.Json(jsonSrcMixed)
	b.ResetTimer()

	for range iter(b.N) {
		dec.Haser()
	}
}

const jsonSrcMixed = `{
	"362ffd": null,
	"df81fe": true,
	"308252": false,
	"f15967": 12,
	"45deb8": -12,
	"eb9b77": 12.34,
	"e70aba": -12.34,
	"214578": 12E34,
	"79eba6": -12e+34,
	"1c8917": "47975f",
	"362ffd": [null],
	"df81fe": [true],
	"308252": [false],
	"f15967": [12],
	"45deb8": [-12],
	"a63f28": [12.34],
	"3c4118": [-12.34],
	"a7b20f": [12E34],
	"1c770b": [-12e+34],
	"ef2de2": ["47975f"],
	"362ffd": [null,     null],
	"df81fe": [true,     true],
	"308252": [false,    false],
	"f15967": [12,       12],
	"45deb8": [-12,      -12],
	"31127a": [12.34,    12.34],
	"c915fe": [-12.34,   -12.34],
	"67e4ac": [12E34,    12E34],
	"c4b04f": [-12e+34,  -12e+34],
	"c8fcd2": ["47975f", "47975f"],
	"362ffd": {"fc087b": null},
	"df81fe": {"eb991f": true},
	"308252": {"e3fb8a": false},
	"f15967": {"c5ee90": 12},
	"45deb8": {"e72825": -12},
	"49674f": {"9cc395": 12.34},
	"13b412": {"e80005": -12.34},
	"953014": {"b2d684": 12E34},
	"fce97c": {"c23603": -12e+34},
	"3b8b71": {"83130e": "47975f"},
	"362ffd": {"811a36": [null]},
	"df81fe": {"2a9a9d": [true]},
	"308252": {"560700": [false]},
	"f15967": {"9f7592": [12]},
	"45deb8": {"c57318": [-12]},
	"56e6e2": {"1ada33": [12.34]},
	"6ddaeb": {"2bb4f6": [-12.34]},
	"6e9237": {"f65105": [12E34]},
	"660e32": {"dc3db8": [-12e+34]},
	"cddc29": {"b30e89": ["47975f"]},
	"362ffd": {"700ac4": [null],     "d8c2dd": [null]},
	"df81fe": {"92f610": [true],     "eeb386": [true]},
	"308252": {"fd0c17": [false],    "4ae24a": [false]},
	"f15967": {"58f388": [12],       "1be428": [12]},
	"45deb8": {"6ac2ee": [-12],      "3e6985": [-12]},
	"cc31e7": {"26af80": [12.34],    "0bad42": [12.34]},
	"da7210": {"e12ae0": [-12.34],   "8cc3ad": [-12.34]},
	"a55030": {"3616fe": [12E34],    "dc0a0c": [12E34]},
	"a79857": {"b762d0": [-12e+34],  "5850fe": [-12e+34]},
	"7b19ef": {"d50fff": ["47975f"], "e833b5": ["47975f"]}
}`
