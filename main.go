package jsondz

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal ...
func Unmarshal(b []byte, intoOneOff ...interface{}) (interface{}, error) {
	res, err := UnmarshalUsingFields(b, intoOneOff...)
	if err != nil {
		res, err = UnmarshalUsingNew(b, intoOneOff...)
	}
	return res, err
}

// UnmarshalUsingNew ...
func UnmarshalUsingNew(b []byte, intoOneOff ...interface{}) (interface{}, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var temp interface{}
	err := d.Decode(&temp)
	if err != nil {
		return nil, err
	}
	m := temp.(map[string]interface{})

	var methodToCall reflect.Value
	var in reflect.Type
	for _, l := range intoOneOff {
		// Check for proper New* functions
		i, met, ok := checkForSingleValueNewFunction(l)
		if !ok {
			continue
		}
		// Check that we can call the new with given json
		match := traverse(reflect.ValueOf(m), i)
		if match {
			if methodToCall.IsValid() {
				return nil, errors.New("Duplicate match found!")
			}
			in = i
			methodToCall = met
		}
	}
	if !methodToCall.IsValid() {
		return nil, errors.New("No match found!")
	}
	ins := reflect.New(in).Interface()

	err = json.Unmarshal(b, &ins)
	if err != nil {
		// Should never happen!
		panic("This should never happen, but somehow this occured: " + err.Error())
	}
	inArray := []reflect.Value{reflect.ValueOf(ins).Elem()}
	res := methodToCall.Call(inArray)

	return res[0].Interface(), nil
}

// UnmarshalUsingFields ...
func UnmarshalUsingFields(b []byte, intoOneOff ...interface{}) (interface{}, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var temp interface{}
	err := d.Decode(&temp)
	if err != nil {
		return nil, err
	}
	m := temp.(map[string]interface{})

	var found interface{}
	for _, l := range intoOneOff {

		match := traverse(reflect.ValueOf(m), reflect.TypeOf(l))
		if match {
			if found != nil {
				return nil, errors.New("Duplicate match found!")
			}
			found = l
		}
	}
	if found == nil {
		return nil, errors.New("No match found!")
	}
	ins := reflect.New(reflect.TypeOf(found)).Interface()
	err = json.Unmarshal(b, &ins)
	if err != nil {
		// Should never happen!
		panic("This should never happen, but somehow this occured: " + err.Error())
	}
	return ins, nil
}

func traverse(v reflect.Value, t reflect.Type) (match bool) {
	switch v.Kind() {
	case reflect.Map:
		// TODO: Logic here bit messy, needs cleaning
		// Idea: fieldNames are must, omitEmpty means that JSON can't have zero val-
		// ues in golang sense for such fields because fields with omit and empty
		// values will not be present in resulting json
		fieldNames, omitEmpty := getJSONFieldNames(t)
		if len(fieldNames) != len(v.MapKeys()) {
			return false
		}
		for _, key := range v.MapKeys() {
			must := fieldNames[key.String()]
			omit := omitEmpty[key.String()]
			value := v.MapIndex(key).Interface()
			if must == "" && omit == "" {
				return false
			}
			if omit != "" {
				if value == nil || isZero(reflect.ValueOf(value)) {
					return false
				}
				must = omit
			}
			f, _ := t.FieldByName(must)
			ok := traverse(reflect.ValueOf(value), f.Type)
			if !ok {
				return false
			}
		}
	case reflect.Slice:
		if t.Kind() != reflect.Slice {
			return false
		}
		trueType := t.Elem()
		for i := 0; i < v.Len(); i++ {
			ok := traverse(reflect.ValueOf(v.Index(i).Interface()), trueType)
			if !ok {
				return false
			}
		}

	case reflect.Bool:
		return t.Kind() == reflect.Bool
	case reflect.String:
		// If number
		var number json.Number
		if v.Type() == reflect.TypeOf(number) {
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				_, err := strconv.ParseInt(v.String(), 10, t.Bits())
				return err == nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				_, err := strconv.ParseUint(v.String(), 10, t.Bits())
				return err == nil
			case reflect.Float32, reflect.Float64:
				_, err := strconv.ParseFloat(v.String(), t.Bits())
				return err == nil
			default:
				return false
			}
		}
		return t.Kind() == reflect.String

	}
	return true
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func getJSONFieldNames(t reflect.Type) (
	fields map[string]string,
	omitEmpty map[string]string) {
	fields = make(map[string]string)
	omitEmpty = make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			tag := field.Tag.Get("json")
			omit := strings.Contains(tag, ",omitempty")

			tag = strings.Replace(tag, ",omitempty", "", 1)

			if tag == "-" {
				continue
			}
			key := field.Name
			if tag != "" {
				key = tag
			}
			if omit {
				omitEmpty[key] = field.Name
			} else {
				fields[key] = field.Name
			}
		} else {
			// Embedded field
			rFields, rOmitEmpty := getJSONFieldNames(field.Type)
			for k, v := range rFields {
				fields[k] = v
			}
			for k, v := range rOmitEmpty {
				omitEmpty[k] = v
			}
		}
	}

	return
}

func checkForSingleValueNewFunction(s interface{}) (in reflect.Type, callMethod reflect.Value, ok bool) {
	// Check that s is indeed a struct
	sv := reflect.ValueOf(s)
	if sv.Kind() != reflect.Struct {
		ok = false
		return
	}

	callMethod = sv.MethodByName("New")
	if !callMethod.IsValid() {
		callMethod = sv.MethodByName("New" + strings.Title(sv.Type().Name()))
		if !callMethod.IsValid() {
			ok = false
			return
		}
	}

	if callMethod.Type().NumIn() != 1 {
		ok = false
		return
	}
	if callMethod.Type().NumOut() != 1 {
		ok = false
		return
	}
	in = callMethod.Type().In(0)
	switch in.Kind() {
	// No support for types below (see http://blog.golang.org/json-and-go)
	case reflect.Func, reflect.Ptr, reflect.Chan, reflect.Complex128, reflect.Complex64:
		ok = false
		return
	}
	out := callMethod.Type().Out(0)
	if out.Kind() != reflect.Ptr {
		ok = false
		return
	}
	if out.Elem() != sv.Type() {
		ok = false
		return
	}
	return in, callMethod, true
}
