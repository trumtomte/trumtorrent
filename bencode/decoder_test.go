package bencode

import (
	_ "fmt"
	"testing"
)

// TODO: more tests, e.g. nested list/dict, decode/encode

func TestBencodedEmptyString(t *testing.T) {
	bytes := make([]byte, 0)
	res, _, err := Decode(bytes)

	if err == nil || res != nil {
		t.Fatal("Should not be able to decode an empty byte array")
	}
}

func TestBencodedString(t *testing.T) {
	data := "5:hello"
	bytes := []byte(data)
	res, _, err := Decode(bytes)

	if err != nil || res != "hello" {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}
}

func TestBencodedInteger(t *testing.T) {
	data := "i42e"
	bytes := []byte(data)
	res, _, err := Decode(bytes)

	if err != nil || res != 42 {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}
}

func TestBencodedNegativeInteger(t *testing.T) {
	data := "i-42e"
	bytes := []byte(data)
	res, _, err := Decode(bytes)

	if err != nil || res != -42 {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}
}

func TestBencodedList(t *testing.T) {
	data := "l5:helloi42e5:worlde"
	bytes := []byte(data)
	res, _, err := Decode(bytes)

	list := res.([]any)
	first := list[0]

	if err != nil || first != "hello" {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}
}

func TestBencodedDictionary(t *testing.T) {
	data := "d5:hello5:worlde"
	bytes := []byte(data)
	res, _, err := Decode(bytes)

	dict, ok := res.(map[string]any)

	if !ok {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}

	value, ok := dict["hello"]

	if err != nil || !ok || value != "world" {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}
}

func TestBencodedDictionaryWithRest(t *testing.T) {
	data := "d5:hello5:worlde"
	bytes := append([]byte(data), byte(255), byte(0), byte(255))
	values, rest, err := Decode(bytes)

	dict, ok := values.(map[string]any)

	if !ok {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}

	value, ok := dict["hello"]

	if err != nil || !ok || value != "world" {
		t.Fatalf("Unable to decode '%v' with reason '%v'", data, err)
	}

	if len(rest) != 3 {
		t.Fatalf("Invalid length of spare bytes (%v)", rest)
	}

	if int(rest[0]) != 255 {
		t.Fatalf("rest[0] != 255 (%v)", rest)
	}
}
