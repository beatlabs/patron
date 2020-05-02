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

// Response definition of the sync Response model.
type Response struct {
	Payload interface{}
	Headers map[string]string
}

// NewResponse creates a new Response.
func NewResponse(p interface{}) *Response {
	return &Response{Payload: p}
}

// ProcessorFunc definition of a function type for processing sync requests.
type ProcessorFunc func(context.Context, *Request) (*Response, error)

// responseReadWriter is a Response writer able to read the Payload.
type responseReadWriter struct {
	buffer     *bytes.Buffer
	len        int
	header     http.Header
	statusCode int
}

// newResponseReadWriter creates a new responseReadWriter.
func newResponseReadWriter() *responseReadWriter {
	return &responseReadWriter{
		buffer: new(bytes.Buffer),
		header: make(http.Header),
	}
}

// read reads the responsereadWriter Payload.
func (rw *responseReadWriter) read(p []byte) (n int, err error) {
	return rw.buffer.Read(p)
}

// readAll returns the Response Payload Bytes.
func (rw *responseReadWriter) readAll() ([]byte, error) {
	if rw.len == 0 {
		// nothing has been written
		return []byte{}, nil
	}
	b := make([]byte, rw.len)
	_, err := rw.read(b)
	return b, err
}

// Header returns the Header object.
func (rw *responseReadWriter) Header() http.Header {
	return rw.header
}

// Write writes the provied Bytes to the byte buffer.
func (rw *responseReadWriter) Write(p []byte) (int, error) {
	rw.len = len(p)
	return rw.buffer.Write(p)
}

// WriteHeader writes the Header status code.
func (rw *responseReadWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
}

func propagateHeaders(header map[string]string, wHeader http.Header) {
	for k, h := range header {
		wHeader.Set(k, h)
	}
}
