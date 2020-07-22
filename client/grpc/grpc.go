// Package grpc provides a client implementation for gRPC with tracing included.
package grpc

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/trace"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	componentName = "grpc-client"
)

// Dial creates a client connection to the given target with a tracing/logging interceptor.
func Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialContext(context.Background(), target, opts...)
}

// DialContext creates a client connection to the given target with a context and a tracing/logging interceptor.
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	if len(opts) == 0 {
		opts = make([]grpc.DialOption, 0)
	}

	opts = append(opts, grpc.WithUnaryInterceptor(unaryInterceptor))

	return grpc.DialContext(ctx, target, opts...)
}

type headersCarrier struct {
	Ctx context.Context
}

// Set implements Set() of opentracing.TextMapWriter.
func (c *headersCarrier) Set(key, val string) {
	c.Ctx = metadata.AppendToOutgoingContext(c.Ctx, key, val)
}

func unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

	span, ctx := trace.ChildSpan(ctx, trace.ComponentOpName(componentName, method), componentName, ext.SpanKindProducer,
		ext.SpanKindProducer)

	carrier := headersCarrier{Ctx: ctx}
	err := span.Tracer().Inject(span.Context(), opentracing.TextMap, &carrier)
	if err != nil {
		return fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	corID := correlation.IDFromContext(carrier.Ctx)

	ctx = metadata.AppendToOutgoingContext(carrier.Ctx, correlation.HeaderID, corID)

	err = invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		trace.SpanError(span)
		return err
	}
	trace.SpanSuccess(span)
	return nil
}
