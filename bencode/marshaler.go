package bencode

import (
	"errors"
	"reflect"
)

func unpackValue(v reflect.Value) any {
	// TODO: add support for types
	switch v.Kind() {
	case reflect.String:
		return v.Interface()
	case reflect.Int:
		return v.Interface()
	case reflect.Struct:
		fieldsByBencodeTag := make(map[string]interface{})

		// NOTE: might wanna start from 1?
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			fieldValue := v.Field(i)
			key := field.Tag.Get("bencode")

			// Skip unintialized values
			if fieldValue.IsZero() {
				continue
			}

			fieldsByBencodeTag[key] = unpackValue(fieldValue)
		}

		return fieldsByBencodeTag
	case reflect.Slice:
		s := make([]any, v.Len())

		for i := 0; i < v.Len(); i++ {
			value := v.Index(i)
			s[i] = unpackValue(value)
		}

		return s
	default:
		return v.Interface()
	}
}

// Marshal takes any value and returns the bencoded data as byte slice
func Marshal(v any) ([]byte, error) {
	value := reflect.ValueOf(v)

	if !value.IsValid() {
		return nil, errors.New("marshal: invalid value")
	}

	// Unpack
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		bencode, _ := Encode(v)
		return bencode, nil
	}

	unpacked := unpackValue(value)
	return Encode(unpacked)
}
