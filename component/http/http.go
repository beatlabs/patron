// Package http provides a HTTP component with included observability.
package http

import (
	"context"
	"io"

	"github.com/beatlabs/patron/encoding"
)

// Request definition of the sync request model.
type Request struct {
	Fields  map[string]string
	Raw     io.Reader
	Headers map[string]string
	decode  encoding.DecodeFunc
}

// NewRequest creates a new request.
func NewRequest(f map[string]string, r io.Reader, h map[string]string, d encoding.DecodeFunc) *Request {
	return &Request{Fields: f, Raw: r, Headers: h, decode: d}
}

// Decode the raw data by using the provided decoder.
func (r *Request) Decode(v interface{}) error {
	return r.decode(r.Raw, v)
}

// Response definition of the sync response model.
type Response struct {
	Payload interface{}
}

// NewResponse creates a new response.
func NewResponse(p interface{}) *Response {
	return &Response{Payload: p}
}

// ProcessorFunc definition of a function type for processing sync requests.
type ProcessorFunc func(context.Context, *Request) (*Response, error)
