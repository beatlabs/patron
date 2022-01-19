package trace

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber/jaeger-client-go"
)

// Counter is a wrapper of a prometheus.Counter.
type Counter struct {
	prometheus.Counter
}

// Add adds the given value to the counter. If there is a span associated with a context ctx the method
// replaces the currently saved exemplar (if any) with a new one, created from the provided value.
func (c *Counter) Add(ctx context.Context, count float64) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			if counter, ok := c.Counter.(prometheus.ExemplarAdder); ok {
				counter.AddWithExemplar(count, prometheus.Labels{TraceID: sctx.TraceID().String()})
				return
			}
		}
	}
	c.Counter.Add(count)
}

// Inc increments the given value to the counter. If there is a span associated with a context ctx the method
// replaces the currently saved exemplar (if any) with a new one, created from the provided value.
func (c *Counter) Inc(ctx context.Context) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			if counter, ok := c.Counter.(prometheus.ExemplarAdder); ok {
				counter.AddWithExemplar(1, prometheus.Labels{TraceID: sctx.TraceID().String()})
				return
			}
		}
	}
	c.Counter.Add(1)
}

// Histogram is a wrapper of a prometheus.Observer.
type Histogram struct {
	prometheus.Observer
}

// Observe adds an observation. If there is a span associated with a context ctx the method replaces
// the currently saved exemplar (if any) with a new one, created from the provided value.
func (h *Histogram) Observe(ctx context.Context, v float64) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			if observer, ok := h.Observer.(prometheus.ExemplarObserver); ok {
				observer.ObserveWithExemplar(v, prometheus.Labels{TraceID: sctx.TraceID().String()})
				return
			}
		}
	}
	h.Observer.Observe(v)
}
