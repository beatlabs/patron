// Package protobuf is a concrete implementation of the encoding abstractions.
package protobuf

import (
	"errors"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/proto"
)

const (
	// Type definition.
	Type string = "application/x-protobuf"
	// TypeGoogle definition.
	TypeGoogle string = "application/x-google-protobuf"
)

// Decode a protobuf input in the form of a reader.
func Decode(data io.Reader, v interface{}) error {
	b, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	return DecodeRaw(b, v)
}

// DecodeRaw a protobuf input in the form of a byte slice.
func DecodeRaw(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errors.New("failed to type assert proto message")
	}
	return proto.Unmarshal(data, msg)
}

// Encode a model to protobuf.
func Encode(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("failed to type assert proto message")
	}
	return proto.Marshal(msg)
}
