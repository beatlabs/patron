// Package encoding provides abstractions for the support of concrete encoding implementations.
package encoding

import (
	"io"
)

const (
	// AcceptHeader definition.
	AcceptHeader string = "Accept"
	// ContentTypeHeader definition.
	ContentTypeHeader string = "Content-Type"
	// ContentEncodingHeader definition.
	ContentEncodingHeader string = "Content-Encoding"
	// ContentLengthHeader definition.
	ContentLengthHeader string = "Content-Length"
	// AcceptEncodingHeader definition, usually a compression algorithm.
	AcceptEncodingHeader string = "Accept-Encoding"
)

// DecodeFunc definition of a decoding function.
type DecodeFunc func(data io.Reader, v interface{}) error

// DecodeRawFunc definition of a decoding function from a byte slice.
type DecodeRawFunc func(data []byte, v interface{}) error

// EncodeFunc definition of an encoding function.
type EncodeFunc func(v interface{}) ([]byte, error)
