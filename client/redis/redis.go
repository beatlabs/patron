// Package redis provides a client with included tracing capabilities.
package redis

import (
	"context"

	"github.com/beatlabs/patron/trace"
	"github.com/go-redis/redis/extra/rediscmd"
	"github.com/go-redis/redis/v8"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	component = "redis"
	dbType    = "kv"
)

// Options wraps redis.Options for easier usage.
type Options redis.Options

// Nil represents the error which is returned in case a key is not found.
const Nil = redis.Nil

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

var _ redis.Hook = tracingHook{}

func (th tracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	_, ctx = startSpan(ctx, th.address, rediscmd.CmdString(cmd))
	return ctx, nil
}

func (th tracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span := opentracing.SpanFromContext(ctx)
	trace.SpanComplete(span, cmd.Err())
	return nil
}

func (th tracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	_, opName := rediscmd.CmdsString(cmds)
	_, ctx = startSpan(ctx, th.address, opName)
	return ctx, nil
}

func (th tracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	span := opentracing.SpanFromContext(ctx)
	trace.SpanComplete(span, cmds[0].Err())
	return nil
}

func startSpan(ctx context.Context, address, opName string) (opentracing.Span, context.Context) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, component)
	ext.DBType.Set(sp, dbType)
	ext.DBInstance.Set(sp, address)
	ext.DBStatement.Set(sp, opName)
	return sp, ctx
}
