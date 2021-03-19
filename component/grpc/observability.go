package grpc

import (
	"context"
	"strings"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	componentName = "gRPC-server"
	unary         = "unary"
	stream        = "stream"
	service       = "service"
	method        = "method"
)

var (
	rpcHandledMetric *prometheus.CounterVec
	rpcLatencyMetric *prometheus.HistogramVec
)

func init() {
	rpcHandledMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "grpc",
			Name:      "handled_total",
			Help:      "Total number of RPC completed on the server.",
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"},
	)
	prometheus.MustRegister(rpcHandledMetric)
	rpcLatencyMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "component",
			Subsystem: "grpc",
			Name:      "handled_seconds",
			Help:      "Latency of a completed RPC on the server.",
		},
		[]string{"grpc_type", "grpc_service", "grpc_method", "grpc_code"})
	prometheus.MustRegister(rpcLatencyMetric)
}

type observer struct {
	typ     string
	corID   string
	service string
	method  string
	sp      opentracing.Span
	ctx     context.Context
	started time.Time
}

func newObserver(ctx context.Context, typ, fullMethodName string) *observer {
	md := grpcMetadata(ctx)
	corID := getCorrelationID(md)

	sp, ctx := grpcSpan(ctx, fullMethodName, corID, md)

	ctx = log.WithContext(ctx, log.Sub(map[string]interface{}{correlation.ID: corID}))

	svc, meth := splitMethodName(fullMethodName)
	return &observer{
		typ:     typ,
		corID:   corID,
		ctx:     ctx,
		method:  meth,
		service: svc,
		sp:      sp,
		started: time.Now(),
	}
}

func (o *observer) observe(err error) {
	dur := time.Since(o.started)
	trace.SpanComplete(o.sp, err)
	o.log(err)
	o.messageHandled(err)
	o.messageLatency(dur, err)
}

func (o *observer) log(err error) {
	if !log.Enabled(log.ErrorLevel) {
		return
	}

	fields := map[string]interface{}{
		"server-type":  "grpc",
		service:        o.service,
		method:         o.method,
		correlation.ID: o.corID,
	}
	if err != nil {
		fields["error"] = err.Error()
		log.Sub(fields).Error()
		return
	}

	log.Sub(fields).Debug()
}

func (o *observer) messageHandled(err error) {
	st, _ := status.FromError(err)
	rpcHandledMetric.WithLabelValues(o.typ, o.service, o.method, st.Code().String()).Inc()
}

func (o *observer) messageLatency(dur time.Duration, err error) {
	st, _ := status.FromError(err)
	rpcLatencyMetric.WithLabelValues(o.typ, o.service, o.method, st.Code().String()).Observe(dur.Seconds())
}

func observableUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	obs := newObserver(ctx, unary, info.FullMethod)
	resp, err = handler(obs.ctx, req)
	obs.observe(err)
	return resp, err
}

func observableStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	obs := newObserver(ss.Context(), stream, info.FullMethod)
	err := handler(srv, ss)
	obs.observe(err)
	return err
}

func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}

func getCorrelationID(md metadata.MD) string {
	values := md.Get(correlation.HeaderID)
	if len(values) == 0 {
		return uuid.New().String()
	}
	return values[0]
}

func mapHeader(md metadata.MD) map[string]string {
	mp := make(map[string]string, md.Len())
	for key, values := range md {
		mp[key] = values[0]
	}
	return mp
}

func grpcSpan(ctx context.Context, fullName, corID string, md metadata.MD) (opentracing.Span, context.Context) {
	return trace.ConsumerSpan(ctx, trace.ComponentOpName(componentName, fullName), componentName,
		corID, mapHeader(md))
}

func grpcMetadata(ctx context.Context) metadata.MD {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(make(map[string]string))
	}
	return md
}
