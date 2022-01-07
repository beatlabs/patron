// Package http provides a client with included tracing capabilities.
package http

import (
	"compress/flate"
	"compress/gzip"
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
	"github.com/beatlabs/patron/log"
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

// Client interface of a HTTP client.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// TracedClient defines a HTTP client with tracing integrated.
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
		nethttp.OperationName(opName(req.Method, req.URL.String())),
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

	return r.(*http.Response), nil
}

func span(path, corID string, r *http.Request) (opentracing.Span, *http.Request) {
	ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract HTTP span: %v", err)
	}
	sp := opentracing.StartSpan(opName(r.Method, path), ext.RPCServerOption(ctx))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.HTTPUrl.Set(sp, r.URL.String())
	ext.Component.Set(sp, clientComponent)
	sp.SetTag(trace.VersionTag, trace.Version)
	sp.SetTag(correlation.ID, corID)
	return sp, r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
}

func finishSpan(sp opentracing.Span, code int) {
	ext.HTTPStatusCode.Set(sp, uint16(code))
	ext.Error.Set(sp, code >= http.StatusInternalServerError)
	sp.Finish()
}

func opName(method, path string) string {
	return method + " " + path
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
