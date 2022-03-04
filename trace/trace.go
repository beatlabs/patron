// Package trace provides trace support and helper methods.
package trace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

const (
	// HostsTag is used to tag the component's hosts.
	HostsTag = "hosts"
	// VersionTag is used to tag the component's version.
	VersionTag = "version"
	// TraceID is a label name for a request trace ID.
	TraceID = "traceID"
)

var (
	cls io.Closer
	// Version will be used to tag all traced components.
	// It can be used to distinguish between dev, stage, and prod environments.
	Version = "dev"
)

// Setup tracing by providing all necessary parameters.
func Setup(name, ver, agent, typ string, prm float64, buckets []float64) error {
	if ver != "" {
		Version = ver
	}
	cfg := config.Configuration{
		ServiceName: name,
		Sampler: &config.SamplerConfig{
			Type:  typ,
			Param: prm,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agent,
		},
	}

	metricsFactory := prometheus.New(
		prometheus.WithBuckets(buckets),
	)
	opts := metrics.NSOptions{Name: name, Tags: nil}
	tr, clsTemp, err := cfg.NewTracer(
		config.Observer(rpcmetrics.NewObserver(metricsFactory.Namespace(opts), rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		return fmt.Errorf("cannot initialize jaeger tracer: %w", err)
	}
	cls = clsTemp
	opentracing.SetGlobalTracer(tr)
	return nil
}

// Close the tracer.
func Close() error {
	log.Debug("closing tracer")
	return cls.Close()
}

// ConsumerSpan starts a new consumer span.
func ConsumerSpan(ctx context.Context, opName, cmp, corID string, hdr map[string]string,
	tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.TextMapCarrier(hdr))
	if err != nil && !errors.Is(err, opentracing.ErrSpanContextNotFound) {
		log.Errorf("failed to extract consumer span: %v", err)
	}
	sp := opentracing.StartSpan(opName, consumerOption{ctx: spCtx})
	ext.Component.Set(sp, cmp)
	sp.SetTag(correlation.ID, corID)
	sp.SetTag(VersionTag, Version)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	return sp, opentracing.ContextWithSpan(ctx, sp)
}

// SpanComplete finishes a span with or without a error indicator.
func SpanComplete(sp opentracing.Span, err error) {
	ext.Error.Set(sp, err != nil)
	sp.Finish()
}

// SpanSuccess finishes a span with a success indicator.
func SpanSuccess(sp opentracing.Span) {
	ext.Error.Set(sp, false)
	sp.Finish()
}

// SpanError finishes a span with a error indicator.
func SpanError(sp opentracing.Span) {
	ext.Error.Set(sp, true)
	sp.Finish()
}

// ChildSpan starts a new child span with specified tags.
func ChildSpan(ctx context.Context, opName, cmp string, tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, cmp)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	sp.SetTag(VersionTag, Version)
	return sp, ctx
}

type consumerOption struct {
	ctx opentracing.SpanContext
}

func (r consumerOption) Apply(o *opentracing.StartSpanOptions) {
	if r.ctx != nil {
		opentracing.ChildOf(r.ctx).Apply(o)
	}
	ext.SpanKindConsumer.Apply(o)
}

// ComponentOpName returns a operation name for a component.
func ComponentOpName(cmp, target string) string {
	return cmp + " " + target
}
