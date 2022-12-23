// Package encoding provides abstractions for supporting concrete encoding implementations.
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

// DecodeFunc definition that supports a reader.
type DecodeFunc func(data io.Reader, v interface{}) error

// DecodeRawFunc definition that supports byte slices.
type DecodeRawFunc func(data []byte, v interface{}) error

// EncodeFunc definition that returns a byte slice.
type EncodeFunc func(v interface{}) ([]byte, error)
