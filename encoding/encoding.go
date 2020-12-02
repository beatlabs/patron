// Package encoding provides abstractions for concrete encoding implementations.
package encoding

import (
	"io"
)

const (
	// AcceptHeader for defining accept encoding.
	AcceptHeader string = "Accept"
	// ContentTypeHeader for defining content type headers.
	ContentTypeHeader string = "Content-Type"
	// ContentEncodingHeader for defining content encoding headers.
	ContentEncodingHeader string = "Content-Encoding"
	// AcceptEncodingHeader for defining accept encoding headers, usually a compression algorithm.
	AcceptEncodingHeader string = "Accept-Encoding"
)

// DecodeFunc function definition of a JSON decoding function.
type DecodeFunc func(data io.Reader, v interface{}) error

// DecodeRawFunc function definition of a JSON decoding function from a byte slice.
type DecodeRawFunc func(data []byte, v interface{}) error

// EncodeFunc function definition of a JSON encoding function.
type EncodeFunc func(v interface{}) ([]byte, error)
