// Package http provides a client with included tracing capabilities.
package http

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"github.com/beatlabs/patron/trace"
)

const (
	clientComponent = "http-client"
)

var reqDurationMetrics *prometheus.HistogramVec

func init() {
	reqDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP requests completed by the client.",
		},
		[]string{"method", "url", "status_code"},
	)
	prometheus.MustRegister(reqDurationMetrics)
}

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
			Transport: &nethttp.Transport{},
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
	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req,
		nethttp.OperationName(opName(req.Method, req.URL.Scheme, req.URL.Host)),
		nethttp.ComponentName(clientComponent))
	defer ht.Finish()

	req.Header.Set(correlation.HeaderID, correlation.IDFromContext(req.Context()))

	start := time.Now()

	rsp, err := tc.do(req)

	ext.HTTPMethod.Set(ht.Span(), req.Method)
	ext.HTTPUrl.Set(ht.Span(), req.URL.String())

	if err != nil {
		ext.Error.Set(ht.Span(), true)
		return rsp, err
	}

	ext.HTTPStatusCode.Set(ht.Span(), uint16(rsp.StatusCode))
	durationHistogram := trace.Histogram{
		Observer: reqDurationMetrics.WithLabelValues(req.Method, req.URL.Host, strconv.Itoa(rsp.StatusCode)),
	}
	durationHistogram.Observe(req.Context(), time.Since(start).Seconds())

	if hdr := req.Header.Get(encoding.AcceptEncodingHeader); hdr != "" {
		rsp.Body = decompress(hdr, rsp)
	}

	return rsp, err
}

func (tc *TracedClient) do(req *http.Request) (*http.Response, error) {
	if tc.cb == nil {
		return tc.cl.Do(req)
	}

	r, err := tc.cb.Execute(func() (interface{}, error) {
		return tc.cl.Do(req)
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

func decompress(hdr string, rsp *http.Response) io.ReadCloser {
	var reader io.ReadCloser
	switch hdr {
	case "gzip":
		reader, _ = gzip.NewReader(rsp.Body)
	case "deflate":
		reader = flate.NewReader(rsp.Body)
	default:
		reader = rsp.Body
	}

	return reader
}
