// Package protobuf is a concrete implementation of the encoding abstractions.
package protobuf

import (
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
	return proto.Unmarshal(data, v.(proto.Message))
}

// Encode a model to protobuf.
func Encode(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}
