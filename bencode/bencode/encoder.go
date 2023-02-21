package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

// Encoder is a simple encoder to encode 'bencoded' data
type Encoder struct {
	buffer *bytes.Buffer
}

// getSortedDictionaryKeys sorts the dictionary keys alphabetically
func (e Encoder) getSortedDictionaryKeys(dict map[string]any) []string {
	i := 0
	keys := make([]string, len(dict))

	for key := range dict {
		keys[i] = key
		i++
	}

	sort.Strings(keys)
	return keys
}

func (e Encoder) consumeString(str string) error {
	size := len(str)
	data := fmt.Sprintf("%v:%v", size, str)

	if _, err := e.buffer.WriteString(data); err != nil {
		return err
	}

	return nil
}

func (e Encoder) consumeInteger(i int) error {
	data := fmt.Sprintf("i%ve", i)

	if _, err := e.buffer.WriteString(data); err != nil {
		return err
	}

	return nil
}

func (e Encoder) consumeList(list []any) error {
	if _, err := e.buffer.WriteString("l"); err != nil {
		return err
	}

	for _, value := range list {
		if err := e.consume(value); err != nil {
			return err
		}
	}

	if _, err := e.buffer.WriteString("e"); err != nil {
		return err
	}

	return nil
}

func (e Encoder) consumeDictionary(dict map[string]any) error {
	if _, err := e.buffer.WriteString("d"); err != nil {
		return err
	}

	// NOTE: according to the spec dictionary keys should be sorted
	keys := e.getSortedDictionaryKeys(dict)

	for _, key := range keys {
		if err := e.consumeString(key); err != nil {
			return err
		}

		value := dict[key]

		if err := e.consume(value); err != nil {
			return err
		}
	}

	if _, err := e.buffer.WriteString("e"); err != nil {
		return err
	}

	return nil
}

func (e Encoder) consume(value any) error {
	// TODO: add support for more types
	switch v := value.(type) {
	case string:
		return e.consumeString(v)
	case int:
		return e.consumeInteger(v)
	case []any:
		return e.consumeList(v)
	case map[string]any:
		return e.consumeDictionary(v)
	default:
		return errors.New("encoder: unable to consume unknown type")
	}
}

func Encode(data any) ([]byte, error) {
	e := &Encoder{buffer: &bytes.Buffer{}}

	if err := e.consume(data); err != nil {
		return nil, err
	}

	return e.buffer.Bytes(), nil
}
