// Package http provides a HTTP component with included observability.
package http

import (
	"context"
	"io"
	"net/http"

	"github.com/beatlabs/patron/encoding"
)

// Header is the http header representation as a map of strings
type Header map[string]string

// Request definition of the sync request model.
type Request struct {
	Fields  map[string]string
	Raw     io.Reader
	Headers Header
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

// Response definition of the sync Response model.
type Response struct {
	Payload interface{}
	Header  Header
}

// NewResponse creates a new Response.
func NewResponse(p interface{}) *Response {
	return &Response{Payload: p, Header: make(map[string]string)}
}

// ProcessorFunc definition of a function type for processing sync requests.
type ProcessorFunc func(context.Context, *Request) (*Response, error)

func propagateHeaders(header Header, wHeader http.Header) {
	for k, h := range header {
		wHeader.Set(k, h)
	}
}
