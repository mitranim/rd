package rd_test

import (
	"fmt"
	"net/url"
	r "reflect"
	"testing"
	"time"

	"github.com/mitranim/rd"
)

func TestParse_num(t *testing.T) {
	test := func(src string, typ r.Type, num int) {
		t.Helper()
		eq(t, num, parseNew(src, typ).Convert(typeInt).Interface())
	}

	for _, typ := range numTypes {
		for _, num := range testNums {
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
			test(fmt.Sprint(num), typ, num)
		}
	}

	for _, typ := range numTypes {
		errs(
			t,
			fmt.Sprintf(`failed to parse "garbage" into %v`, typ),
			rd.Parse(`garbage`, r.New(typ).Elem()),
		)
	}
}

func TestParse_bool(t *testing.T) {
	testOk := func(exp bool, src string) {
		t.Helper()
		eq(t, exp, parseNew(src, typeBool).Bool())
	}

	testOk(true, `true`)
	testOk(false, `false`)

	testFail := func(src string) {
		t.Helper()
		errs(
			t,
			fmt.Sprintf(`failed to parse %q into bool`, src),
			rd.Parse(src, r.New(typeBool).Elem()),
		)
	}

	testFail(``)
	testFail(`garbage`)
	testFail(`0`)
	testFail(`1`)
	testFail(`True`)
	testFail(`False`)
	testFail(`TRUE`)
	testFail(`FALSE`)
	testFail(`t`)
	testFail(`f`)
	testFail(`yes`)
	testFail(`no`)
	testFail(`on`)
	testFail(`off`)
}

func TestParse_string(t *testing.T) {
	test := func(src string) {
		t.Helper()
		eq(t, src, parseNew(src, typeString).String())
	}

	test(``)
	test(`1e25d8acb91f425ea645d7049f6592e1`)
	test(`f0c1ea163f6f4d839b74889438dcb1d5`)
}

func TestParse_bytes(t *testing.T) {
	test := func(src string) {
		t.Helper()
		eq(t, []byte(src), parseNew(src, typeBytes).Bytes())
	}

	test(``)
	test(`1e25d8acb91f425ea645d7049f6592e1`)
	test(`f0c1ea163f6f4d839b74889438dcb1d5`)
}

func TestParse_unmarshaler(t *testing.T) {
	testOk := func(src string, exp time.Time) {
		t.Helper()
		eq(t, exp, parseNew(src, typeTime).Interface())
	}

	testOk(`0001-01-01T00:00:00Z`, time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))
	testOk(`1234-01-02T03:04:05Z`, time.Date(1234, 1, 2, 3, 4, 5, 0, time.UTC))

	testParseFail(t, ``, typeTime, `cannot parse`)
	testParseFail(t, `garbage`, typeTime, `cannot parse`)
}

func TestParse_parser(t *testing.T) {
	testOk := func(src string, exp TimeParser) {
		t.Helper()
		eq(t, exp, parseNew(src, typeTimeParser).Interface())
	}

	testOk(`0001-01-01T00:00:00Z`, TimeParser(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)))
	testOk(`1234-01-02T03:04:05Z`, TimeParser(time.Date(1234, 1, 2, 3, 4, 5, 0, time.UTC)))

	testParseFail(t, ``, typeTimeParser, `cannot parse`)
	testParseFail(t, `garbage`, typeTimeParser, `cannot parse`)
}

func testParseFail(t testing.TB, src string, typ r.Type, msg string) {
	t.Helper()
	errs(
		t,
		fmt.Sprintf(`%v %q`, msg, src),
		rd.Parse(src, r.New(typeTime).Elem()),
	)
}

// `rd.ParseSlice` delegates to `rd.Parse` which is tested separately.
// Here we mainly need to verify slice handling.
func TestParseSlice(t *testing.T) {
	test := func(exp []int, src []string, tar []int) {
		t.Helper()
		try(rd.ParseSlice(src, r.ValueOf(&tar).Elem()))
		eq(t, exp, tar)
	}

	test([]int(nil), []string(nil), []int(nil))
	test([]int(nil), []string(nil), []int{})
	test([]int(nil), []string(nil), []int{10, 20})
	test([]int{}, []string{}, []int(nil))
	test([]int{}, []string{}, []int{})
	test([]int{}, []string{}, []int{10, 20})
	test([]int{30}, []string{`30`}, []int(nil))
	test([]int{30}, []string{`30`}, []int{})
	test([]int{30}, []string{`30`}, []int{10, 20})
	test([]int{30, 40}, []string{`30`, `40`}, []int(nil))
	test([]int{30, 40}, []string{`30`, `40`}, []int{})
	test([]int{30, 40}, []string{`30`, `40`}, []int{10, 20})
}

func TestParseSlice_SliceParser(t *testing.T) {
	var tar SliceParserStruct
	try(rd.ParseSlice([]string{`10`, `20`}, r.ValueOf(&tar).Elem()))
	eq(t, SliceParserStruct{[]int{10, 20}}, tar)
}

// Incomplete, needs more test cases.
func TestJson_Haser_parsing(t *testing.T) {
	test := func(exp rd.Set, src string) {
		t.Helper()
		eq(t, exp, rd.Json(src).Haser())
	}

	test(set(), ``)
	test(set(), `                `)
	test(set(), `null`)
	test(set(), `true`)
	test(set(), `false`)
	test(set(), `10`)
	test(set(), `"str"`)
	test(set(), `[]`)
	test(set(), ` [ ] `)
	test(set(), `[null, true, false, 10, "one", {"two": "three"}]`)
	test(set(), `[{"one": "two"}]`)
	test(set(), `{}`)
	test(set(), ` { } `)
	test(set(), `arbitrary garbage`)

	test(set(`one`), `{"one": null}`)
	test(set(`one`), `{"one": true}`)
	test(set(`one`), `{"one": false}`)
	test(set(`one`), `{"one": 0}`)
	test(set(`one`), `{"one": -0}`)
	test(set(`one`), `{"one": 1}`)
	test(set(`one`), `{"one": -1}`)
	test(set(`one`), `{"one": 12}`)
	test(set(`one`), `{"one": -12}`)
	test(set(`one`), `{"one": 1.2}`)
	test(set(`one`), `{"one": -1.2}`)
	test(set(`one`), `{"one": 12.3}`)
	test(set(`one`), `{"one": -12.3}`)
	test(set(`one`), `{"one": 1.23}`)
	test(set(`one`), `{"one": -1.23}`)
	test(set(`one`), `{"one": 12.34}`)
	test(set(`one`), `{"one": -12.34}`)
	test(set(`one`), `{"one": 1.2E3}`)
	test(set(`one`), `{"one": 1.2E34}`)
	test(set(`one`), `{"one": 1.2e3}`)
	test(set(`one`), `{"one": 1.2e34}`)
	test(set(`one`), `{"one": 1.2E+3}`)
	test(set(`one`), `{"one": 1.2E+34}`)
	test(set(`one`), `{"one": 1.2e+3}`)
	test(set(`one`), `{"one": 1.2e+34}`)
	test(set(`one`), `{"one": 1.2E-3}`)
	test(set(`one`), `{"one": 1.2E-34}`)
	test(set(`one`), `{"one": 1.2e-3}`)
	test(set(`one`), `{"one": 1.2e-34}`)
	test(set(`one`), `{"one": 1.23E4}`)
	test(set(`one`), `{"one": 1.23E45}`)
	test(set(`one`), `{"one": 1.23e4}`)
	test(set(`one`), `{"one": 1.23e45}`)
	test(set(`one`), `{"one": 1.23E+4}`)
	test(set(`one`), `{"one": 1.23E+45}`)
	test(set(`one`), `{"one": 1.23e+4}`)
	test(set(`one`), `{"one": 1.23e+45}`)
	test(set(`one`), `{"one": 1.23E-4}`)
	test(set(`one`), `{"one": 1.23E-45}`)
	test(set(`one`), `{"one": 1.23e-4}`)
	test(set(`one`), `{"one": 1.23e-45}`)
	test(set(`one`), `{"one": -1.2E3}`)
	test(set(`one`), `{"one": -1.2E34}`)
	test(set(`one`), `{"one": -1.2e3}`)
	test(set(`one`), `{"one": -1.2e34}`)
	test(set(`one`), `{"one": -1.2E+3}`)
	test(set(`one`), `{"one": -1.2E+34}`)
	test(set(`one`), `{"one": -1.2e+3}`)
	test(set(`one`), `{"one": -1.2e+34}`)
	test(set(`one`), `{"one": -1.2E-3}`)
	test(set(`one`), `{"one": -1.2E-34}`)
	test(set(`one`), `{"one": -1.2e-3}`)
	test(set(`one`), `{"one": -1.2e-34}`)
	test(set(`one`), `{"one": -1.23E4}`)
	test(set(`one`), `{"one": -1.23E45}`)
	test(set(`one`), `{"one": -1.23e4}`)
	test(set(`one`), `{"one": -1.23e45}`)
	test(set(`one`), `{"one": -1.23E+4}`)
	test(set(`one`), `{"one": -1.23E+45}`)
	test(set(`one`), `{"one": -1.23e+4}`)
	test(set(`one`), `{"one": -1.23e+45}`)
	test(set(`one`), `{"one": -1.23E-4}`)
	test(set(`one`), `{"one": -1.23E-45}`)
	test(set(`one`), `{"one": -1.23e-4}`)
	test(set(`one`), `{"one": -1.23e-45}`)
	test(set(`one`), `{"one": "two"}`)
	test(set(`one`), `{"one": ["two"]}`)
	test(set(`one`), `{"one": {"two": "three"}}`)
	test(set(`one`, `two`), `{"one": null, "two": null}`)
	test(set(`one`, `two`), `{"one": true, "two": true}`)
	test(set(`one`, `two`), `{"one": false, "two": false}`)
	test(set(`one`, `two`), `{"one": 10, "two": 20}`)
	test(set(`one`, `two`), `{"one": "three", "two": "four"}`)
	test(set(`one`, `two`), `{"one": ["three"], "two" : ["four"]}`)
	test(set(`one`, `two`), `{"one": ["three", "four"], "two": ["five", "six"]}`)
	test(set(`one`, `two`), `{"one": {"three\\four": "five\\six"}, "two" : { "seven" : [ "eight" , "nine" ] } }`)
	test(set(`one\\two`, `two\\three`), `{"one\\two": null, "two\\three": null}`)

	// TODO test panics on invalid syntax.
}

func TestJson_Haser(t *testing.T) {
	eq(t, testOuterJsonSet, rd.Json(testOuterJson).Haser())
}

func TestJson_Set(t *testing.T) {
	eq(t, testOuterJsonSet, rd.Json(testOuterJson).Set())
}

func TestForm_Haser(t *testing.T) {
	val := rd.Form(testOuterQuery)
	eq(t, val, val.Haser())
}

func TestForm_Set(t *testing.T) {
	eq(t, testOuterQuerySet, rd.Form(testOuterQuery).Set())
}

func TestForm_Has(t *testing.T) {
	haser := rd.Form(testOuterQuery)

	eq(t, true, haser.Has(`embedStr`))
	eq(t, true, haser.Has(`embedNum`))
	eq(t, true, haser.Has(`outerStr`))
	eq(t, false, haser.Has(``))
	eq(t, false, haser.Has(`inner`))
	eq(t, false, haser.Has(`innerStr`))
	eq(t, false, haser.Has(`innerNum`))
}

// `rd.Json.Decode` delegates to `json.Unmarshal`.
// We only need to verify that it does, in fact, unmarshal.
func TestJson_Decode(t *testing.T) {
	var tar Outer
	try(rd.Json(testOuterJson).Decode(&tar))
	eq(t, testOuter, tar)
}

func TestForm_Decode(t *testing.T) {
	test := func(t testing.TB, exp, tar interface{}, src url.Values) {
		testDec(t, exp, tar, rd.Form(src))
	}

	t.Run(`normal`, func(t *testing.T) {
		test(t, TarUnusable{}, TarUnusable{}, unusableVals)
		test(t, TarVoid{}, TarVoid{}, url.Values{})
		test(t, TarVoid{}, TarVoid{}, url.Values{`one`: {`two`}})

		for _, src := range nopQueries {
			test(t, TarInt{10}, TarInt{10}, src)
			test(t, TarPtrInt{ptrInt(10)}, TarPtrInt{ptrInt(10)}, src)
			test(t, TarSliceInt{[]int{10, 20}}, TarSliceInt{[]int{10, 20}}, src)
			test(t, TarSlicePtrInt{[]*int{ptrInt(10), ptrInt(20)}}, TarSlicePtrInt{[]*int{ptrInt(10), ptrInt(20)}}, src)
		}

		for _, src := range zeroQueries {
			test(t, TarInt{}, TarInt{10}, src)
			test(t, TarPtrInt{}, TarPtrInt{ptrInt(10)}, src)
			test(t, TarSliceInt{}, TarSliceInt{[]int{10, 20}}, src)
			test(t, TarSlicePtrInt{}, TarSlicePtrInt{[]*int{ptrInt(10), ptrInt(20)}}, src)
		}

		for i, src := range numQueries {
			test(t, TarInt{20}, TarInt{10}, src)
			test(t, TarPtrInt{ptrInt(20)}, TarPtrInt{ptrInt(10)}, src)
			test(t, TarSliceInt{numOutputs[i]}, TarSliceInt{[]int{10, 20}}, src)
			test(t, TarSlicePtrInt{numPtrOutputs[i]}, TarSlicePtrInt{[]*int{ptrInt(10), ptrInt(20)}}, src)
		}
	})

	t.Run(`missing fields are unaffected`, func(t *testing.T) {
		test(
			t,
			TarPair{One: []int{10, 20}, Two: []int{30, 40}},
			TarPair{One: []int{10, 20}, Two: []int{30, 40}},
			url.Values{`three`: {`50`, `60`}},
		)

		test(
			t,
			TarPair{One: []int{10, 20}, Two: []int{50, 60}},
			TarPair{One: []int{10, 20}, Two: []int{30, 40}},
			url.Values{`two`: {`50`, `60`}},
		)

		test(
			t,
			TarPair{One: []int{50, 60}, Two: []int{30, 40}},
			TarPair{One: []int{10, 20}, Two: []int{30, 40}},
			url.Values{`one`: {`50`, `60`}},
		)
	})

	t.Run(`structs embedded by value`, func(t *testing.T) {
		test(
			t,
			testOuterSimple,
			Outer{
				Embed:    Embed{EmbedStr: `embed val old`, EmbedNum: 20},
				OuterStr: `outer val old`,
			},
			testOuterQuery,
		)
	})

	t.Run(`structs embedded by pointer`, func(t *testing.T) {
		test(
			t,
			testPtrOuterSimple,
			PtrOuter{OuterStr: `outer val old`},
			testOuterQuery,
		)
	})

	t.Run(`invokes SliceParser`, func(t *testing.T) {
		type T struct {
			One   SliceParserStruct `json:"one"`
			Two   SliceParserStruct `json:"two"`
			Three SliceParserStruct `json:"three"`
		}

		test(
			t,
			T{
				One:   SliceParserStruct{[]int{70, 80}},
				Two:   SliceParserStruct{[]int{90, 100}},
				Three: SliceParserStruct{[]int{50, 60}},
			},
			T{
				One:   SliceParserStruct{[]int{10, 20}},
				Two:   SliceParserStruct{[]int{30, 40}},
				Three: SliceParserStruct{[]int{50, 60}},
			},
			url.Values{
				`one`: {`70`, `80`},
				`two`: {`90`, `100`},
			},
		)
	})
}

func testDec(t testing.TB, exp, tar interface{}, dec rd.Dec) {
	t.Helper()

	ptr := r.New(r.TypeOf(tar))
	ptr.Elem().Set(r.ValueOf(tar))

	try(dec.Decode(ptr.Interface()))
	eq(t, exp, ptr.Elem().Interface())
}

func TestForm_Parse(t *testing.T) {
	var tar rd.Form
	try(tar.Parse(testOuterQuery.Encode()))
	eq(t, url.Values(tar), testOuterQuery)
}

func TestDecode_GET_query(t *testing.T) {
	req := Req{}.Query(testOuterQuery).Ptr()

	var tar Outer
	rd.TryDecode(req, &tar)

	eq(t, testOuterSimple, tar)
}

func TestDecode_GET_json(t *testing.T) {
	req := Req{}.BodyJson(testOuterJson).Ptr()

	var tar Outer
	rd.TryDecode(req, &tar)

	eq(t, testOuter, tar)
}

func TestDecode_POST_json(t *testing.T) {
	req := Req{}.Post().BodyJson(testOuterJson).Ptr()

	var tar Outer
	rd.TryDecode(req, &tar)

	eq(t, testOuter, tar)
}

func TestDecode_POST_form(t *testing.T) {
	req := Req{}.Post().Query(testUrlQuery).BodyForm(testOuterQuery).Ptr()

	var tar Outer
	rd.TryDecode(req, &tar)

	eq(t, testOuterSimple, tar)
}

func TestDecode_POST_multi(t *testing.T) {
	req := Req{}.Post().Query(testUrlQuery).BodyMulti(testOuterQuery).Ptr()

	var tar Outer
	rd.TryDecode(req, &tar)

	eq(t, testOuterSimple, tar)
}

func TestDownload_GET_query(t *testing.T) {
	req := Req{}.Query(testOuterQuery).Ptr()
	eq(t, rd.Form(req.URL.Query()), rd.TryDownload(req))
}

func TestDownload_POST_query(t *testing.T) {
	req := Req{}.Post().Query(testOuterQuery).Ptr()
	eq(t, rd.Form(req.URL.Query()), rd.TryDownload(req))
}

func TestDownload_POST_json(t *testing.T) {
	req := Req{}.Post().Query(testUrlQuery).BodyJson(testJsonStr).Ptr()
	eq(t, rd.Json(testJsonStr), rd.TryDownload(req))
}

func TestDownload_POST_form(t *testing.T) {
	req := Req{}.Post().Query(testUrlQuery).BodyForm(testBodyQuery).Ptr()
	eq(t, rd.Form(testBodyQuery), rd.TryDownload(req))
}

func TestDownload_POST_multi(t *testing.T) {
	req := Req{}.Post().Query(testUrlQuery).BodyMulti(testBodyQuery).Ptr()
	eq(t, rd.Form(testBodyQuery), rd.TryDownload(req))
}
