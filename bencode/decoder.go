package bencode

import (
	"bufio"
	"bytes"
	"errors"
	_ "fmt"
	"io"
	"strconv"
)

// Decoder is a simple decoder for 'bencoded' data
type Decoder struct {
	reader *bufio.Reader
}

func (d Decoder) skip() error {
	_, err := d.reader.ReadByte()
	return err
}

func (d Decoder) peek() (byte, error) {
	bytes, err := d.reader.Peek(1)
	if err != nil {
		return 0, err
	}

	return bytes[0], nil
}

func (d Decoder) consumeString() (string, error) {
	sizeStr, err := d.reader.ReadString(':')
	if err != nil {
		return "", err
	}

	size, err := strconv.Atoi(sizeStr[:len(sizeStr)-1])
	if err != nil {
		return "", err
	}

	if size < 0 {
		return "", errors.New("decoder: negative string length not allowed")
	}

	if size == 0 {
		return "", nil
	}

	buf := make([]byte, size)

	if _, err := io.ReadFull(d.reader, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func (d Decoder) consumeInteger() (int, error) {
	iStr, err := d.reader.ReadString('e')
	if err != nil {
		return 0, err
	}

	if len(iStr) < 3 {
		return 0, errors.New("decoder: integer string is too short")
	}

	if len(iStr) > 3 {
		if iStr[1] == '-' && iStr[2] == '0' {
			return 0, errors.New("decoder: integers cannot start with -0")
		}

		if iStr[1] == '0' && iStr[2] != 'e' {
			return 0, errors.New("decoder: integers cannot start with 0")
		}
	}

	i, err := strconv.Atoi(iStr[1 : len(iStr)-1])
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (d Decoder) consumeList() ([]any, error) {
	var list []any

	if err := d.skip(); err != nil {
		return nil, err
	}

	for {
		char, err := d.peek()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if char == 'e' {
			break
		}

		item, err := d.consume()
		if err != nil {
			return nil, err
		}

		list = append(list, item)
	}

	if err := d.skip(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d Decoder) consumeDictionary() (map[string]any, error) {
	dict := make(map[string]any)

	if err := d.skip(); err != nil {
		return nil, err
	}

	for {
		char, err := d.peek()

		if err == io.EOF {
			// NOTE: investigate why `break` wasn't needed here...
		}

		if err != nil {
			return nil, err
		}

		if char == 'e' {
			break
		}

		key, err := d.consumeString()
		if err != nil {
			return nil, err
		}

		value, err := d.consume()
		if err != nil {
			return nil, err
		}

		dict[key] = value
	}

	if err := d.skip(); err != nil {
		return nil, err
	}

	return dict, nil
}

func (d Decoder) consume() (any, error) {
	char, err := d.peek()
	if err != nil {
		return nil, err
	}

	switch char {
	case 'l':
		return d.consumeList()
	case 'd':
		return d.consumeDictionary()
	case 'i':
		return d.consumeInteger()
	default:
		return d.consumeString()
	}
}

func Decode(data []byte) (values any, rest []byte, err error) {
	d := &Decoder{reader: bufio.NewReader(bytes.NewReader(data))}

	if values, err = d.consume(); err != nil {
		return
	}

	rest, err = io.ReadAll(d.reader)
	return
}
