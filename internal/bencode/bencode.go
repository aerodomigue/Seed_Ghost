package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

var (
	ErrInvalid    = errors.New("bencode: invalid data")
	ErrUnexpected = errors.New("bencode: unexpected end of data")
)

// Decode parses bencoded data and returns the decoded value.
// Possible return types: int64, string, []interface{}, map[string]interface{}
func Decode(data []byte) (interface{}, error) {
	r := &reader{data: data, pos: 0}
	val, err := decodeValue(r)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// DecodeBytes is like Decode but reads from a byte slice and also returns bytes consumed.
func DecodeBytes(data []byte) (interface{}, int, error) {
	r := &reader{data: data, pos: 0}
	val, err := decodeValue(r)
	if err != nil {
		return nil, 0, err
	}
	return val, r.pos, nil
}

// Encode serializes a value into bencoded format.
func Encode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type reader struct {
	data []byte
	pos  int
}

func (r *reader) peek() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, ErrUnexpected
	}
	return r.data[r.pos], nil
}

func (r *reader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, ErrUnexpected
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *reader) readBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, ErrUnexpected
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func decodeValue(r *reader) (interface{}, error) {
	b, err := r.peek()
	if err != nil {
		return nil, err
	}
	switch {
	case b == 'i':
		return decodeInt(r)
	case b == 'l':
		return decodeList(r)
	case b == 'd':
		return decodeDict(r)
	case b >= '0' && b <= '9':
		return decodeString(r)
	default:
		return nil, fmt.Errorf("%w: unexpected byte %q at position %d", ErrInvalid, b, r.pos)
	}
}

func decodeInt(r *reader) (int64, error) {
	r.readByte() // consume 'i'
	var numBytes []byte
	for {
		b, err := r.readByte()
		if err != nil {
			return 0, err
		}
		if b == 'e' {
			break
		}
		numBytes = append(numBytes, b)
	}
	if len(numBytes) == 0 {
		return 0, fmt.Errorf("%w: empty integer", ErrInvalid)
	}
	// Leading zeros not allowed (except i0e)
	if len(numBytes) > 1 && numBytes[0] == '0' {
		return 0, fmt.Errorf("%w: leading zero in integer", ErrInvalid)
	}
	if len(numBytes) > 1 && numBytes[0] == '-' && numBytes[1] == '0' {
		return 0, fmt.Errorf("%w: negative zero", ErrInvalid)
	}
	n, err := strconv.ParseInt(string(numBytes), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return n, nil
}

func decodeString(r *reader) (string, error) {
	raw, err := decodeStringBytes(r)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeStringBytes(r *reader) ([]byte, error) {
	var lenBytes []byte
	for {
		b, err := r.readByte()
		if err != nil {
			return nil, err
		}
		if b == ':' {
			break
		}
		if b < '0' || b > '9' {
			return nil, fmt.Errorf("%w: invalid string length character %q", ErrInvalid, b)
		}
		lenBytes = append(lenBytes, b)
	}
	length, err := strconv.Atoi(string(lenBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid string length", ErrInvalid)
	}
	return r.readBytes(length)
}

func decodeList(r *reader) ([]interface{}, error) {
	r.readByte() // consume 'l'
	var list []interface{}
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.readByte()
			return list, nil
		}
		val, err := decodeValue(r)
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}
}

func decodeDict(r *reader) (map[string]interface{}, error) {
	r.readByte() // consume 'd'
	dict := make(map[string]interface{})
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.readByte()
			return dict, nil
		}
		key, err := decodeString(r)
		if err != nil {
			return nil, fmt.Errorf("dict key: %w", err)
		}
		val, err := decodeValue(r)
		if err != nil {
			return nil, fmt.Errorf("dict value for key %q: %w", key, err)
		}
		dict[key] = val
	}
}

func encodeValue(w io.Writer, v interface{}) error {
	switch val := v.(type) {
	case int:
		return encodeInt(w, int64(val))
	case int64:
		return encodeInt(w, val)
	case string:
		return encodeString(w, val)
	case []byte:
		return encodeBytes(w, val)
	case []interface{}:
		return encodeList(w, val)
	case map[string]interface{}:
		return encodeDict(w, val)
	default:
		return fmt.Errorf("bencode: unsupported type %T", v)
	}
}

func encodeInt(w io.Writer, n int64) error {
	_, err := fmt.Fprintf(w, "i%de", n)
	return err
}

func encodeString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "%d:%s", len(s), s)
	return err
}

func encodeBytes(w io.Writer, b []byte) error {
	_, err := fmt.Fprintf(w, "%d:", len(b))
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func encodeList(w io.Writer, l []interface{}) error {
	if _, err := w.Write([]byte{'l'}); err != nil {
		return err
	}
	for _, v := range l {
		if err := encodeValue(w, v); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte{'e'})
	return err
}

func encodeDict(w io.Writer, d map[string]interface{}) error {
	if _, err := w.Write([]byte{'d'}); err != nil {
		return err
	}
	// Keys must be sorted
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := encodeString(w, k); err != nil {
			return err
		}
		if err := encodeValue(w, d[k]); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte{'e'})
	return err
}

// DecodeRaw decodes bencoded data preserving byte strings as []byte.
// This is important for info_hash calculation where we need raw bytes.
func DecodeRaw(data []byte) (interface{}, error) {
	r := &reader{data: data, pos: 0}
	return decodeValueRaw(r)
}

func decodeValueRaw(r *reader) (interface{}, error) {
	b, err := r.peek()
	if err != nil {
		return nil, err
	}
	switch {
	case b == 'i':
		return decodeInt(r)
	case b == 'l':
		return decodeListRaw(r)
	case b == 'd':
		return decodeDictRaw(r)
	case b >= '0' && b <= '9':
		return decodeStringBytes(r)
	default:
		return nil, fmt.Errorf("%w: unexpected byte %q at position %d", ErrInvalid, b, r.pos)
	}
}

func decodeListRaw(r *reader) ([]interface{}, error) {
	r.readByte()
	var list []interface{}
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.readByte()
			return list, nil
		}
		val, err := decodeValueRaw(r)
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}
}

func decodeDictRaw(r *reader) (map[string]interface{}, error) {
	r.readByte()
	dict := make(map[string]interface{})
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.readByte()
			return dict, nil
		}
		keyBytes, err := decodeStringBytes(r)
		if err != nil {
			return nil, err
		}
		val, err := decodeValueRaw(r)
		if err != nil {
			return nil, err
		}
		dict[string(keyBytes)] = val
	}
}

// ExtractRawValue extracts the raw bencoded bytes for a given key in a bencoded dict.
// This is used to get the raw "info" dict for hashing.
func ExtractRawValue(data []byte, key string) ([]byte, error) {
	r := &reader{data: data, pos: 0}
	b, err := r.readByte()
	if err != nil || b != 'd' {
		return nil, fmt.Errorf("%w: expected dict", ErrInvalid)
	}
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			return nil, fmt.Errorf("key %q not found", key)
		}
		k, err := decodeString(r)
		if err != nil {
			return nil, err
		}
		startPos := r.pos
		// Skip over the value to find its end
		_, err = decodeValueRaw(r)
		if err != nil {
			return nil, err
		}
		endPos := r.pos
		if k == key {
			return data[startPos:endPos], nil
		}
	}
}
