package bencode

import (
	"testing"
)

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"i42e", 42, false},
		{"i0e", 0, false},
		{"i-1e", -1, false},
		{"i1234567890e", 1234567890, false},
		{"i00e", 0, true},  // leading zero
		{"i-0e", 0, true},  // negative zero
		{"ie", 0, true},    // empty
		{"i12", 0, true},   // no end
	}
	for _, tt := range tests {
		val, err := Decode([]byte(tt.input))
		if tt.wantErr {
			if err == nil {
				t.Errorf("Decode(%q) expected error, got %v", tt.input, val)
			}
			continue
		}
		if err != nil {
			t.Errorf("Decode(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if val.(int64) != tt.expected {
			t.Errorf("Decode(%q) = %v, want %v", tt.input, val, tt.expected)
		}
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"4:spam", "spam", false},
		{"0:", "", false},
		{"3:foo", "foo", false},
		{"5:hello", "hello", false},
		{"10:0123456789", "0123456789", false},
		{"4:spa", "", true},   // too short
		{"x:foo", "", true},   // invalid length
	}
	for _, tt := range tests {
		val, err := Decode([]byte(tt.input))
		if tt.wantErr {
			if err == nil {
				t.Errorf("Decode(%q) expected error, got %v", tt.input, val)
			}
			continue
		}
		if err != nil {
			t.Errorf("Decode(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if val.(string) != tt.expected {
			t.Errorf("Decode(%q) = %q, want %q", tt.input, val, tt.expected)
		}
	}
}

func TestDecodeList(t *testing.T) {
	val, err := Decode([]byte("l4:spami42ee"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := val.([]interface{})
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0].(string) != "spam" {
		t.Errorf("list[0] = %q, want %q", list[0], "spam")
	}
	if list[1].(int64) != 42 {
		t.Errorf("list[1] = %v, want 42", list[1])
	}
}

func TestDecodeEmptyList(t *testing.T) {
	val, err := Decode([]byte("le"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := val.([]interface{})
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestDecodeDict(t *testing.T) {
	val, err := Decode([]byte("d3:bar4:spam3:fooi42ee"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict := val.(map[string]interface{})
	if dict["bar"].(string) != "spam" {
		t.Errorf("dict[bar] = %q, want %q", dict["bar"], "spam")
	}
	if dict["foo"].(int64) != 42 {
		t.Errorf("dict[foo] = %v, want 42", dict["foo"])
	}
}

func TestDecodeEmptyDict(t *testing.T) {
	val, err := Decode([]byte("de"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict := val.(map[string]interface{})
	if len(dict) != 0 {
		t.Errorf("expected empty dict, got %d keys", len(dict))
	}
}

func TestDecodeNested(t *testing.T) {
	// d4:listl4:testi1ee4:name3:fooe
	val, err := Decode([]byte("d4:listl4:testi1ee4:name3:fooe"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dict := val.(map[string]interface{})
	list := dict["list"].([]interface{})
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if dict["name"].(string) != "foo" {
		t.Errorf("name = %q, want %q", dict["name"], "foo")
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{int64(42), "i42e"},
		{int64(0), "i0e"},
		{int64(-1), "i-1e"},
		{"spam", "4:spam"},
		{"", "0:"},
		{[]interface{}{"spam", int64(42)}, "l4:spami42ee"},
		{[]interface{}{}, "le"},
		{map[string]interface{}{"bar": "spam", "foo": int64(42)}, "d3:bar4:spam3:fooi42ee"},
	}
	for _, tt := range tests {
		data, err := Encode(tt.input)
		if err != nil {
			t.Errorf("Encode(%v) unexpected error: %v", tt.input, err)
			continue
		}
		if string(data) != tt.expected {
			t.Errorf("Encode(%v) = %q, want %q", tt.input, string(data), tt.expected)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"announce": "http://tracker.example.com/announce",
		"info": map[string]interface{}{
			"name":         "test.txt",
			"piece length": int64(262144),
			"length":       int64(1024),
		},
	}
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	dict := decoded.(map[string]interface{})
	if dict["announce"].(string) != "http://tracker.example.com/announce" {
		t.Errorf("announce mismatch")
	}
	info := dict["info"].(map[string]interface{})
	if info["name"].(string) != "test.txt" {
		t.Errorf("info.name mismatch")
	}
}

func TestExtractRawValue(t *testing.T) {
	input := "d4:infod4:name4:teste8:announce3:urle"
	raw, err := ExtractRawValue([]byte(input), "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "d4:name4:teste"
	if string(raw) != expected {
		t.Errorf("ExtractRawValue = %q, want %q", string(raw), expected)
	}
}

func TestDecodeInvalidData(t *testing.T) {
	invalids := []string{
		"",
		"x",
		"i",
		"1",
		"d3:foo",
	}
	for _, input := range invalids {
		_, err := Decode([]byte(input))
		if err == nil {
			t.Errorf("Decode(%q) expected error", input)
		}
	}
}
