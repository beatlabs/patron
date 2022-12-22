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

// Decode a reader input into a model.
func Decode(data io.Reader, v interface{}) error {
	return json.NewDecoder(data).Decode(v)
}

// DecodeRaw by taking byte slice input into a model.
func DecodeRaw(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Encode a model to protobuf and return a byte slice.
func Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
