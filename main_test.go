package jsondz

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

type Example struct {
	String string `json:"name"`
	Int    int    `json:"age"`
}

type NestedExample struct {
	IntArray   []int `json:"IntArray"`
	FloatArray []float64
	Omit       chan<- struct{} `json:"-"`
	Nested     []Example
	Bool       bool `json:"Bool"`
}

type NestedExampleExtended struct {
	NestedExample
	Extended string
}

func TestBasicNestedExample(t *testing.T) {
	example := NestedExample{
		IntArray:   []int{2, 1, 3, 4, 5},
		FloatArray: []float64{3.0, -4.0},
		Omit:       nil,
		Nested:     []Example{{"Jack", 50}, {"", 28}},
		Bool:       true,
	}
	runSingleTest(t, example, NestedExample{}, NestedExampleExtended{})
}

func convert(in interface{}, options ...interface{}) (c interface{}, o interface{}, err error) {
	b, err := json.Marshal(&in)
	if err != nil {
		return nil, nil, err
	}
	res, err := UnmarshalExactMatch(b, options...)
	if err != nil {
		return nil, nil, err
	}
	// Copy of in to be used in unmarshal of original data
	nin := reflect.New(reflect.TypeOf(in)).Interface()
	err = json.Unmarshal(b, &nin)

	if err != nil {
		return nil, nil, err
	}
	return res, nin, nil
}

func runSingleTest(t *testing.T, origin interface{}, options ...interface{}) {
	actual, expected, err := convert(origin, options...)
	if err != nil {
		t.Error("Failed due to ", err)
		t.FailNow()
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Actual (%s) and Expected (%s) should be the same!\n",
			actual, expected)
		t.FailNow()
	}
}

type Numbers struct {
	F64  float64
	F32  float32
	I64  int64
	I32  int32
	I16  int16
	I8   int8
	UI64 uint64
	UI32 uint32
	UI16 uint16
	UI8  uint8
}

func TestDifferentSizeFloatsInts(t *testing.T) {
	max := Numbers{
		math.MaxFloat64,
		math.MaxFloat32,
		math.MaxInt64,
		math.MaxInt32,
		math.MaxInt16,
		math.MaxInt8,
		math.MaxUint64,
		math.MaxUint32,
		math.MaxUint16,
		math.MaxUint8,
	}
	runSingleTest(t, max, Numbers{})
	min := Numbers{
		-math.MaxFloat64,
		-math.MaxFloat32,
		-math.MaxInt64,
		-math.MaxInt32,
		-math.MaxInt16,
		-math.MaxInt8,
		0,
		0,
		0,
		0,
	}
	runSingleTest(t, min, Numbers{})
	epsNeg := Numbers{
		-math.SmallestNonzeroFloat64,
		-math.SmallestNonzeroFloat32,
		-1,
		-1,
		-1,
		-1,
		0,
		0,
		0,
		0,
	}
	runSingleTest(t, epsNeg, Numbers{})

	epsPos := Numbers{
		math.SmallestNonzeroFloat64,
		math.SmallestNonzeroFloat32,
		1,
		1,
		1,
		1,
		0,
		0,
		0,
		0,
	}
	runSingleTest(t, epsPos, Numbers{})
}

func TestOverflowF32(t *testing.T) {
	s := struct{ F32 float64 }{math.MaxFloat32 * 2}
	mapTo := struct{ F32 float32 }{}
	_, _, err := convert(s, mapTo)
	if err == nil {
		t.Fail()
	}
}

func TestOverflowI32(t *testing.T) {
	s := struct{ I32 int64 }{math.MaxInt32 * 2}
	mapTo := struct{ I32 int32 }{}
	_, _, err := convert(s, mapTo)
	if err == nil {
		t.Fail()
	}
}

func TestUnderflowF32(t *testing.T) {
	s := struct{ F32 float64 }{-math.MaxFloat32 * 2}
	mapTo := struct{ F32 float32 }{}
	_, _, err := convert(s, mapTo)
	if err == nil {
		t.Fail()
	}
}

func TestUnderflowI32(t *testing.T) {
	s := struct{ I32 int64 }{-math.MaxInt32 * 2}
	mapTo := struct{ I32 int32 }{}
	_, _, err := convert(s, mapTo)
	if err == nil {
		t.Fail()
	}
}

func TestOverflowUI32(t *testing.T) {
	s := struct{ UI32 uint64 }{math.MaxUint32 * 2}
	mapTo := struct{ UI32 uint32 }{}
	_, _, err := convert(s, mapTo)
	if err == nil {
		t.Fail()
	}
}

func TestDuplicate(t *testing.T) {
	a := struct {
		A string
		B string
	}{A: "A", B: "B"}
	b := struct {
		A string
		B string
	}{}

	_, _, err := convert(a, a, b)
	if err == nil {
		t.Fail()
	}
}

func TestSliceInJsonNotInTarget(t *testing.T) {
	a := struct{ A []string }{A: []string{"A", "B"}}
	b := struct{ A string }{}
	_, _, err := convert(a, b)
	if err == nil {
		t.Fail()
	}
}

func TestSliceInJsonDifferentTypeInTarget(t *testing.T) {
	a := struct{ A []string }{A: []string{"A", "B"}}
	b := struct{ A []int }{}
	_, _, err := convert(a, b)
	if err == nil {
		t.Fail()
	}
}

func TestDifferentTypes(t *testing.T) {
	a := struct{ A string }{"A"}
	b := struct{ A int }{6}
	_, _, err := convert(a, b)
	if err == nil {
		t.Fail()
	}
	_, _, err = convert(b, a)
	if err == nil {
		t.Fail()
	}

}

func TestDifferentKeys(t *testing.T) {
	a := struct {
		A string
		B string
	}{"A", "B"}
	b := struct {
		A string
		C string
	}{"A", "C"}
	_, _, err := convert(a, b)
	if err == nil {
		t.Fail()
	}
	_, _, err = convert(b, a)
	if err == nil {
		t.Fail()
	}

}

func TestBrokenJson(t *testing.T) {
	example := "{\"item\":\"endquote\"}"
	_, err := UnmarshalExactMatch([]byte(example), struct{ item string }{})
	if err != nil {
		t.Fail()
	}
	example = "{\"item\":\"noendquote}"
	_, err = UnmarshalExactMatch([]byte(example), struct{ item string }{})
	if err == nil {
		t.Fail()
	}
}

func TestJsonOmitEmpty(t *testing.T) {
	jsn := `{"Field1":"","Field2":"A"}`
	s := struct {
		Field1 string `json:",omitempty"`
		Field2 string
	}{"", "A"}
	result, err := UnmarshalExactMatch([]byte(jsn), s)
	if err == nil {
		t.Error(err)
		t.FailNow()
	}

	jsn1 := `{"Field2":"A"}`
	s1 := struct {
		Field1 string `json:",omitempty"`
		Field2 string
	}{"", "A"}
	result, err = UnmarshalExactMatch([]byte(jsn1), s1)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if !reflect.DeepEqual(result, &s1) {
		t.Errorf("Actual (%s) and Expected (%s) should be the same!\n",
			result, s)
		t.FailNow()
	}

}

func TestJsonOmitNilArray(t *testing.T) {
	jsn := `{"Field1":null,"Field2":"A"}`
	s := struct {
		Field1 []int `json:",omitempty"`
		Field2 string
	}{[]int{}, "A"}
	_, err := UnmarshalExactMatch([]byte(jsn), s)
	if err == nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestJsonOmitNilStruct(t *testing.T) {
	jsn := `{"Field1":null,"Field2":"A"}`
	s := struct {
		Field1 struct{} `json:",omitempty"`
		Field2 string
	}{struct{}{}, "A"}
	_, err := UnmarshalExactMatch([]byte(jsn), s)
	if err == nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestIsZero(t *testing.T) {
	if !isZero(reflect.ValueOf(struct{}{})) {
		t.FailNow()
	}
	var m map[string]string
	if !isZero(reflect.ValueOf(m)) {
		t.FailNow()
	}
	var a [4]int
	if !isZero(reflect.ValueOf(a)) {
		t.FailNow()
	}
}

type node struct {
	Embedded
	A string
}

type Embedded struct {
	B string
}

func TestEmbeddedStructs(t *testing.T) {
	a := node{A: "A", Embedded: Embedded{"B"}}
	runSingleTest(t, a, node{})
}
