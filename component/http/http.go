package http

import (
	"bytes"
	"context"
	"io"
	"net/http"

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
	Headers map[string]string
}

// NewResponse creates a new response.
func NewResponse(p interface{}) *Response {
	return &Response{Payload: p}
}

// ProcessorFunc definition of a function type for processing sync requests.
type ProcessorFunc func(context.Context, *Request) (*Response, error)

// ResponseReadWriter is a response writer able to read the payload.
type ResponseReadWriter struct {
	buffer     *bytes.Buffer
	len        int
	header     http.Header
	statusCode int
}

// NewResponseReadWriter creates a new ResponseReadWriter.
func NewResponseReadWriter() *ResponseReadWriter {
	return &ResponseReadWriter{
		buffer: new(bytes.Buffer),
		header: make(http.Header),
	}
}

// Read reads the ResponsereadWriter payload.
func (rw *ResponseReadWriter) Read(p []byte) (n int, err error) {
	return rw.buffer.Read(p)
}

// ReadAll returns the response payload bytes.
func (rw *ResponseReadWriter) ReadAll() ([]byte, error) {
	if rw.len == 0 {
		// nothing has been written
		return []byte{}, nil
	}
	b := make([]byte, rw.len)
	_, err := rw.Read(b)
	return b, err
}

// Header returns the Header object.
func (rw *ResponseReadWriter) Header() http.Header {
	return rw.header
}

// Write writes the provied bytes to the byte buffer.
func (rw *ResponseReadWriter) Write(p []byte) (int, error) {
	rw.len = len(p)
	return rw.buffer.Write(p)
}

// WriteHeader writes the header status code.
func (rw *ResponseReadWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
}

func propagateHeaders(header map[string]string, wHeader http.Header) {
	for k, h := range header {
		wHeader.Set(k, h)
	}
}
