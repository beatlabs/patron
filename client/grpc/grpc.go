// Package grpc provides a client implementation for gRPC with tracing and
// metrics included.
package grpc

import (
	"context"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	componentName = "grpc-client"
	unary         = "unary"
)

var (
	rpcDurationMetrics *prometheus.HistogramVec
)

func init() {
	rpcDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "grpc",
			Name:      "rpc_duration_seconds",
			Help:      "RPC requests completed by the client.",
		},
		[]string{"grpc_type", "grpc_target", "grpc_method", "grpc_code"})

	prometheus.MustRegister(rpcDurationMetrics)
}

// Dial creates a client connection to the given target with a tracing and
// metrics unary interceptor.
func Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialContext(context.Background(), target, opts...)
}

// DialContext creates a client connection to the given target with a context and
// a tracing and metrics unary interceptor.
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	if len(opts) == 0 {
		opts = make([]grpc.DialOption, 0)
	}

	opts = append(opts, grpc.WithUnaryInterceptor(unaryInterceptor(target)))

	return grpc.DialContext(ctx, target, opts...)
}

type headersCarrier struct {
	Ctx context.Context
}

// Set implements Set() of opentracing.TextMapWriter.
func (c *headersCarrier) Set(key, val string) {
	c.Ctx = metadata.AppendToOutgoingContext(c.Ctx, key, val)
}

func unaryInterceptor(target string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		span, ctx := trace.ChildSpan(ctx,
			trace.ComponentOpName(componentName, method),
			componentName,
			ext.SpanKindProducer,
		)
		carrier := headersCarrier{Ctx: ctx}
		err := span.Tracer().Inject(span.Context(), opentracing.TextMap, &carrier)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to inject tracing headers: %v", err)
		}

		corID := correlation.IDFromContext(carrier.Ctx)
		ctx = metadata.AppendToOutgoingContext(carrier.Ctx, correlation.HeaderID, corID)
		invokeTime := time.Now()
		err = invoker(ctx, method, req, reply, cc, opts...)
		invokeDuration := time.Since(invokeTime)

		rpcStatus, _ := status.FromError(err) // codes.OK if err == nil, codes.Unknown if !ok

		durationHistogram := trace.Histogram{
			Observer: rpcDurationMetrics.WithLabelValues(unary, target, method, rpcStatus.Code().String()),
		}
		durationHistogram.Observe(ctx, invokeDuration.Seconds())

		if err != nil {
			trace.SpanError(span)
			return err
		}

		trace.SpanSuccess(span)
		return nil
	}
}
