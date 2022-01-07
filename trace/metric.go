package trace

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber/jaeger-client-go"
)

type Counter struct {
	prometheus.Counter
}

type Histogram struct {
	prometheus.Observer
}

type CounterOperation interface {
	Inc(ctx context.Context)
	Add(ctx context.Context, val float64)
}

type HistogramOperation interface {
	Observe(ctx context.Context, v float64)
}

func (c *Counter) Add(ctx context.Context, count float64) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			c.Counter.(prometheus.ExemplarAdder).AddWithExemplar(
				count, prometheus.Labels{TraceID: sctx.TraceID().String()},
			)
		} else {
			c.Counter.Add(count)
		}
	} else {
		c.Counter.Add(count)
	}
}

func (c *Counter) Inc(ctx context.Context) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			c.Counter.(prometheus.ExemplarAdder).AddWithExemplar(
				1, prometheus.Labels{TraceID: sctx.TraceID().String()},
			)
		} else {
			c.Counter.Inc()
		}
	} else {
		c.Counter.Inc()
	}
}

func (h *Histogram) Observe(ctx context.Context, v float64) {
	spanFromCtx := opentracing.SpanFromContext(ctx)
	if spanFromCtx != nil {
		if sctx, ok := spanFromCtx.Context().(jaeger.SpanContext); ok {
			h.Observer.(prometheus.ExemplarObserver).ObserveWithExemplar(
				v, prometheus.Labels{TraceID: sctx.TraceID().String()},
			)
		} else {
			h.Observer.Observe(v)
		}
	} else {
		h.Observer.Observe(v)
	}
}
