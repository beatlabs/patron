package trace

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

const (
	// KafkaConsumerComponent definition.
	KafkaConsumerComponent = "kafka-consumer"
	// KafkaAsyncProducerComponent definition.
	KafkaAsyncProducerComponent = "kafka-async-producer"
	// AMQPConsumerComponent definition.
	AMQPConsumerComponent = "amqp-consumer"
	// AMQPPublisherComponent definition.
	AMQPPublisherComponent = "amqp-publisher"
	// HTTPComponent definition.
	HTTPComponent = "http"
	// HTTPClientComponent definition.
	HTTPClientComponent = "http-client"
	// SQSConsumerComponent definition.
	SQSConsumerComponent = "sqs-consumer"
	// SNSPublisherComponent definition.
	SNSPublisherComponent = "sns-publisher"
	versionTag            = "version"
	hostsTag              = "hosts"
)

var (
	cls     io.Closer
	version = "dev"
)

// Setup tracing by providing all necessary parameters.
func Setup(name, ver, agent, typ string, prm float64) error {
	if ver != "" {
		version = ver
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
	time.Sleep(100 * time.Millisecond)
	metricsFactory := prometheus.New()
	tr, clsTemp, err := cfg.NewTracer(
		config.Logger(jaegerLoggerAdapter{}),
		config.Observer(rpcmetrics.NewObserver(metricsFactory.Namespace(name, nil), rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		return fmt.Errorf("cannot initialize jaeger tracer: %w", err)
	}
	cls = clsTemp
	opentracing.SetGlobalTracer(tr)
	version = ver
	return nil
}

// Close the tracer.
func Close() error {
	log.Debug("closing tracer")
	return cls.Close()
}

// HTTPSpan starts a new HTTP span.
func HTTPSpan(path, corID string, r *http.Request) (opentracing.Span, *http.Request) {
	ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract HTTP span: %v", err)
	}
	sp := opentracing.StartSpan(HTTPOpName(r.Method, path), ext.RPCServerOption(ctx))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.HTTPUrl.Set(sp, r.URL.String())
	ext.Component.Set(sp, "http")
	sp.SetTag(versionTag, version)
	sp.SetTag(correlation.ID, corID)
	return sp, r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
}

// FinishHTTPSpan finishes a HTTP span by providing a HTTP status code.
func FinishHTTPSpan(sp opentracing.Span, code int) {
	ext.HTTPStatusCode.Set(sp, uint16(code))
	ext.Error.Set(sp, code >= http.StatusInternalServerError)
	sp.Finish()
}

// ConsumerSpan starts a new consumer span.
func ConsumerSpan(ctx context.Context, opName, cmp, corID string, hdr map[string]string,
	tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.TextMapCarrier(hdr))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract consumer span: %v", err)
	}
	sp := opentracing.StartSpan(opName, consumerOption{ctx: spCtx})
	ext.Component.Set(sp, cmp)
	sp.SetTag(correlation.ID, corID)
	sp.SetTag(versionTag, version)
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
	sp.SetTag(versionTag, version)
	return sp, ctx
}

// SQLSpan starts a new SQL child span with specified tags.
func SQLSpan(ctx context.Context, opName, cmp, sqlType, instance, user, stmt string,
	tags ...opentracing.Tag) (opentracing.Span, context.Context) {

	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, cmp)
	ext.DBType.Set(sp, sqlType)
	ext.DBInstance.Set(sp, instance)
	ext.DBUser.Set(sp, user)
	ext.DBStatement.Set(sp, stmt)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	sp.SetTag(versionTag, version)
	return sp, ctx
}

// EsSpan starts a new elasticsearch child span with specified tags.
func EsSpan(ctx context.Context, opName, cmp, user, uri, method, body string, hostPool []string) opentracing.Span {

	sp, _ := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, cmp)
	ext.DBType.Set(sp, "elasticsearch")
	ext.DBUser.Set(sp, user)

	ext.HTTPUrl.Set(sp, uri)
	ext.HTTPMethod.Set(sp, method)
	ext.DBStatement.Set(sp, body)

	hostsFmt := "[" + strings.Join(hostPool, ", ") + "]"
	sp.SetTag(hostsTag, hostsFmt)
	sp.SetTag(versionTag, version)

	return sp
}

// HTTPOpName return a string representation of the HTTP request operation.
func HTTPOpName(method, path string) string {
	return method + " " + path
}

type jaegerLoggerAdapter struct {
}

func (l jaegerLoggerAdapter) Error(msg string) {
	log.Error(msg)
}

func (l jaegerLoggerAdapter) Infof(msg string, args ...interface{}) {
	log.Infof(msg, args...)
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
