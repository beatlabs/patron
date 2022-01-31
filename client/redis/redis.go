// Package redis provides a client with included tracing capabilities.
package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/beatlabs/patron/trace"
	"github.com/go-redis/redis/extra/rediscmd"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	component = "redis"
	dbType    = "kv"
	// Nil represents the error which is returned in case a key is not found.
	Nil = redis.Nil
)

var (
	cmdDurationMetrics *prometheus.HistogramVec
	_                  redis.Hook = tracingHook{}
)

func init() {
	cmdDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "redis",
			Name:      "cmd_duration_seconds",
			Help:      "Redis commands completed by the client.",
		},
		[]string{"command", "success"},
	)
	prometheus.MustRegister(cmdDurationMetrics)
}

type duration struct{}

// Options wraps redis.Options for easier usage.
type Options redis.Options

// Client represents a connection with a Redis client.
type Client struct {
	redis.Client
}

// New returns a new Redis client.
func New(opt Options) Client {
	clientOptions := redis.Options(opt)
	cl := redis.NewClient(&clientOptions)
	cl.AddHook(tracingHook{address: cl.Options().Addr})
	return Client{Client: *cl}
}

type tracingHook struct {
	address string
}

func (th tracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	_, ctx = startSpan(ctx, th.address, rediscmd.CmdString(cmd))
	return context.WithValue(ctx, duration{}, time.Now()), nil
}

func (th tracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span := opentracing.SpanFromContext(ctx)
	trace.SpanComplete(span, cmd.Err())
	observeDuration(ctx, rediscmd.CmdString(cmd), cmd.Err())
	return nil
}

func (th tracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	_, opName := rediscmd.CmdsString(cmds)
	_, ctx = startSpan(ctx, th.address, opName)
	return context.WithValue(ctx, duration{}, time.Now()), nil
}

func (th tracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	span := opentracing.SpanFromContext(ctx)
	trace.SpanComplete(span, cmds[0].Err())
	_, opName := rediscmd.CmdsString(cmds)
	observeDuration(ctx, opName, cmds[0].Err())
	return nil
}

func observeDuration(ctx context.Context, cmd string, err error) {
	dur := time.Since(ctx.Value(duration{}).(time.Time))
	durationHistogram := trace.Histogram{
		Observer: cmdDurationMetrics.WithLabelValues(cmd, strconv.FormatBool(err == nil)),
	}
	durationHistogram.Observe(ctx, dur.Seconds())
}

func startSpan(ctx context.Context, address, opName string) (opentracing.Span, context.Context) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, component)
	ext.DBType.Set(sp, dbType)
	ext.DBInstance.Set(sp, address)
	ext.DBStatement.Set(sp, opName)
	return sp, ctx
}
