// Package protobuf is a concrete implementation of the encoding abstractions.
package protobuf

import (
	"errors"
	"io"

	"google.golang.org/protobuf/proto"
)

const (
	// Type definition.
	Type string = "application/x-protobuf"
	// TypeGoogle definition.
	TypeGoogle string = "application/x-google-protobuf"
)

// Decode a reader input into a protobuf model.
func Decode(data io.Reader, v interface{}) error {
	b, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	return DecodeRaw(b, v)
}

// DecodeRaw a byte slice input into a protobuf model.
func DecodeRaw(data []byte, v interface{}) error {
	val, ok := v.(proto.Message)
	if !ok {
		return errors.New("failed to type assert to proto message")
	}
	return proto.Unmarshal(data, val)
}

// Encode a protobuf model into a byte slice.
func Encode(v interface{}) ([]byte, error) {
	val, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("failed to type assert to proto message")
	}
	return proto.Marshal(val)
}
