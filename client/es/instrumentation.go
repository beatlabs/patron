package es

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName       = "es"
	endpointAttrKey   = "endpoint"
	metricName        = "es.request.duration"
	metricDescription = "Elasticsearch request duration."
)

var (
	errStatusCode     = errors.New("elasticsearch request failed")
	durationHistogram metric.Int64Histogram
)

func init() {
	durationHistogram = patronmetric.Int64Histogram(packageName, metricName, metricDescription, "ms")
}

// otelMetricInstrumentation wraps the upstream OpenTelemetry instrumentation to add metrics.
type otelMetricInstrumentation struct {
	delegate elastictransport.Instrumentation
}

type stateKey struct{}

type requestState struct {
	startTime  time.Time
	endpoint   string
	statusCode int
	err        error
}

func (st *requestState) reset() {
	st.startTime = time.Time{}
	st.endpoint = ""
	st.statusCode = 0
	st.err = nil
}

// newMetricInstrumentation creates an instrumentation that records OTEL metrics
// and delegates tracing to the upstream implementation.
func newMetricInstrumentation(version string) elastictransport.Instrumentation {
	return &otelMetricInstrumentation{
		delegate: elastictransport.NewOtelInstrumentation(nil, false, version),
	}
}

func (o *otelMetricInstrumentation) Start(ctx context.Context, name string) context.Context {
	ctx = o.delegate.Start(ctx, name)
	st := requestStatePool.Get().(*requestState)
	st.reset()
	st.startTime = time.Now()
	st.endpoint = name
	return context.WithValue(ctx, stateKey{}, st)
}

func (o *otelMetricInstrumentation) Close(ctx context.Context) {
	// Record metrics before ending the span
	if st, ok := ctx.Value(stateKey{}).(*requestState); ok && !st.startTime.IsZero() {
		attrSet := attribute.NewSet(
			observability.ClientAttribute(packageName),
			attribute.String(endpointAttrKey, st.endpoint),
			observability.StatusAttribute(st.finalErr()),
		)
		durationHistogram.Record(ctx, time.Since(st.startTime).Milliseconds(), metric.WithAttributeSet(attrSet))
		st.reset()
		requestStatePool.Put(st)
	}
	o.delegate.Close(ctx)
}

func (o *otelMetricInstrumentation) RecordError(ctx context.Context, err error) {
	if st, ok := ctx.Value(stateKey{}).(*requestState); ok {
		st.err = err
	}
	o.delegate.RecordError(ctx, err)
}

func (o *otelMetricInstrumentation) RecordPathPart(ctx context.Context, pathPart, value string) {
	o.delegate.RecordPathPart(ctx, pathPart, value)
}

func (o *otelMetricInstrumentation) RecordRequestBody(ctx context.Context, endpoint string, query io.Reader) io.ReadCloser {
	return o.delegate.RecordRequestBody(ctx, endpoint, query)
}

func (o *otelMetricInstrumentation) BeforeRequest(req *http.Request, endpoint string) {
	o.delegate.BeforeRequest(req, endpoint)
}

func (o *otelMetricInstrumentation) AfterRequest(req *http.Request, system, endpoint string) {
	o.delegate.AfterRequest(req, system, endpoint)
}

func (o *otelMetricInstrumentation) AfterResponse(ctx context.Context, res *http.Response) {
	if st, ok := ctx.Value(stateKey{}).(*requestState); ok && res != nil {
		st.statusCode = res.StatusCode
	}
	o.delegate.AfterResponse(ctx, res)
}

func (st *requestState) finalErr() error {
	if st.err != nil {
		return st.err
	}
	if st.statusCode >= 400 {
		return errStatusCode
	}
	return nil
}

var requestStatePool = sync.Pool{New: func() any { return &requestState{} }}
