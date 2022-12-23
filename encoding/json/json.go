// Package json is a concrete implementation of the encoding abstractions.
package json

import (
	"encoding/json"
	"io"
)

const (
	// Type JSON definition.
	Type string = "application/json"
	// TypeCharset JSON definition with charset.
	TypeCharset string = "application/json; charset=utf-8"
)

// Decode a reader input into a JSON model.
func Decode(data io.Reader, v interface{}) error {
	return json.NewDecoder(data).Decode(v)
}

// DecodeRaw a byte slice input into a JSON model.
func DecodeRaw(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Encode a JSON model and return a byte slice.
func Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
