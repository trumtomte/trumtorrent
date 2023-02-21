package bencode

import (
	"errors"
	"reflect"
)

func unmarshalSlice(t reflect.Type, v reflect.Value) reflect.Value {
	slice := reflect.ValueOf(v.Interface())
	sliceT := reflect.MakeSlice(t, slice.Len(), slice.Cap())
	itemT := sliceT.Type().Elem()

	for i := 0; i < slice.Len(); i++ {
		value := slice.Index(i)
		valueT := unmarshalByType(itemT, value)
		sliceT.Index(i).Set(valueT)
	}

	return sliceT
}

func unmarshalMap(t reflect.Type, v reflect.Value) reflect.Value {
	mapT := reflect.MakeMap(t)
	itemT := mapT.Type().Elem()
	iter := reflect.ValueOf(v.Interface()).MapRange()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		valueT := unmarshalByType(itemT, value)
		mapT.SetMapIndex(key, valueT)
	}

	return mapT
}

func unmarshalStruct(t reflect.Type, v reflect.Value) reflect.Value {
	structT := reflect.New(t).Elem()
	iter := reflect.ValueOf(v.Interface()).MapRange()

	for iter.Next() {
		tag := iter.Key().String()

		fieldT := structT.FieldByNameFunc(func(name string) bool {
			if field, ok := structT.Type().FieldByName(name); ok {
				return field.Tag.Get("bencode") == tag
			} else {
				return false
			}
		})

		// Silently skip invalid or private fields
		if !fieldT.IsValid() || !fieldT.CanSet() {
			continue
		}

		value := iter.Value()
		valueT := unmarshalByType(fieldT.Type(), value)
		fieldT.Set(valueT)
	}

	return structT
}

func unmarshalByType(t reflect.Type, v reflect.Value) reflect.Value {
	// TODO: add support for more types
	switch t.Kind() {
	case reflect.Slice:
		return unmarshalSlice(t, v)
	case reflect.Map:
		return unmarshalMap(t, v)
	case reflect.Struct:
		return unmarshalStruct(t, v)
	case reflect.String:
		return reflect.ValueOf(v.Interface())
	case reflect.Int:
		return reflect.ValueOf(v.Interface())
	default:
		// FIXME: this naively ignores the type and just takes the value
		return reflect.ValueOf(v.Interface())
	}
}

// Unmarshal takes a bencoded byte slice and unpacks it unto a struct
func Unmarshal(data []byte, v any) error {
	values, _, err := Decode(data)
	if err != nil {
		return err
	}

	bencodeV := reflect.ValueOf(values)

	if !bencodeV.IsValid() {
		return errors.New("unmarshal: data is not valid")
	}

	if bencodeV.Kind() != reflect.Map {
		return errors.New("unmarshal: unable to decode the bencoded data into a map")
	}

	structV := reflect.ValueOf(v)

	if !structV.IsValid() {
		return errors.New("unmarshal: 'v' is not valid")
	}

	if structV.Kind() == reflect.Pointer {
		structV = structV.Elem()
	}

	if structV.Kind() != reflect.Struct {
		return errors.New("unmarshal: 'v' is not a struct")
	}

	value := unmarshalByType(structV.Type(), bencodeV)
	structV.Set(value)

	return nil
}
