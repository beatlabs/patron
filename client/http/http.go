// Package http provides a client with included tracing capabilities.
package http

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client interface of an HTTP client.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// TracedClient defines an HTTP client with tracing integrated.
type TracedClient struct {
	cl *http.Client
	cb *circuitbreaker.CircuitBreaker
}

// New creates a new HTTP client.
func New(oo ...OptionFunc) (*TracedClient, error) {
	tc := &TracedClient{
		cl: &http.Client{
			Timeout:   60 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		cb: nil,
	}

	for _, o := range oo {
		err := o(tc)
		if err != nil {
			return nil, err
		}
	}

	return tc, nil
}

// Do execute an HTTP request with integrated tracing and tracing propagation downstream.
func (tc *TracedClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set(correlation.HeaderID, correlation.IDFromContext(req.Context()))

	rsp, err := tc.do(req)
	if err != nil {
		return rsp, err
	}

	if hdr := req.Header.Get(encoding.AcceptEncodingHeader); hdr != "" {
		rsp.Body, err = decompress(hdr, rsp)
		if err != nil {
			return rsp, err
		}
	}

	return rsp, err
}

func (tc *TracedClient) do(req *http.Request) (*http.Response, error) {
	if tc.cb == nil {
		return tc.cl.Do(req)
	}

	r, err := tc.cb.Execute(func() (any, error) {
		return tc.cl.Do(req) // nolint:bodyclose
	})
	if err != nil {
		return nil, err
	}

	rsp, ok := r.(*http.Response)
	if !ok {
		return nil, errors.New("failed to type assert to response")
	}

	return rsp, nil
}

func opName(method, scheme, host string) string {
	return method + " " + scheme + "://" + host
}

func decompress(hdr string, rsp *http.Response) (io.ReadCloser, error) {
	switch hdr {
	case "gzip":
		return gzip.NewReader(rsp.Body)
	case "deflate":
		return flate.NewReader(rsp.Body), nil
	default:
		return rsp.Body, nil
	}
}
