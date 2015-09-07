package jsondz

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// UnmarshalExactMatch ...
func UnmarshalExactMatch(b []byte, lookUp ...interface{}) (interface{}, error) {
	// Parse json to anonymous map
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var f interface{}
	err := d.Decode(&f)
	if err != nil {
		return nil, err
	}
	fmt.Println(f)
	fmt.Println(string(b))
	m := f.(map[string]interface{})

	fmt.Println("RAWMAP: ", reflect.ValueOf(m).MapKeys())
	var found interface{}
	for _, l := range lookUp {
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
		panic("This should never happen, but somehow :" + err.Error()) // Should never happen!
	}
	return ins, nil
}

func traverse(v reflect.Value, t reflect.Type) (match bool) {
	// Check if map, thus Obj in JSON
	switch v.Kind() {
	case reflect.Map:
		fmt.Println("TYPE: ", v.Type())

		fmt.Println("MAP ", v, t)

		fieldNames, omitEmpty := getJSONFieldNames(t)
		fmt.Println("MAP KEYS :", v.MapKeys(), " FIELD NAMES: ", fieldNames, len(fieldNames))
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

			fmt.Println("MAP PROCESS, KEY: ", key, ", FOUND: ", must)

			f, _ := t.FieldByName(must)
			ok := traverse(reflect.ValueOf(value), f.Type)
			if !ok {
				return false
			}
		}
	case reflect.Slice:
		fmt.Println("TYPE: ", v.Type())

		fmt.Println("SLICE! ", v, t)

		if t.Kind() != reflect.Slice {
			return false
		}
		trueType := t.Elem()
		fmt.Println("TRUETYPE: ", trueType.Kind())
		for i := 0; i < v.Len(); i++ {
			ok := traverse(reflect.ValueOf(v.Index(i).Interface()), trueType)
			if !ok {
				return false
			}
		}

	case reflect.Bool:
		fmt.Println("TYPE: ", v.Type())

		return t.Kind() == reflect.Bool
	case reflect.String:
		// If number
		var number json.Number
		number = ""
		if v.Type() == reflect.TypeOf(number) {
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fmt.Println("INT:", v, t.Bits())
				_, err := strconv.ParseInt(v.String(), 10, t.Bits())
				return err == nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fmt.Println("UINT:", v, t.Bits())
				_, err := strconv.ParseUint(v.String(), 10, t.Bits())
				return err == nil
			case reflect.Float32, reflect.Float64:
				fmt.Println("FLOAT:", v, t.Bits())
				_, err := strconv.ParseFloat(v.String(), t.Bits())
				fmt.Println(err)
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

func getJSONFieldNames(t reflect.Type) (fields map[string]string, omitEmpty map[string]string) {
	fields = make(map[string]string)
	omitEmpty = make(map[string]string)
	fmt.Println("FIELD NAMES FOR ", t)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			tag := field.Tag.Get("json")
			omit := strings.Contains(tag, ",omitempty")

			tag = strings.Replace(tag, ",omitempty", "", 1)
			fmt.Println("TAG:", tag)

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
		}
	}

	return
}
