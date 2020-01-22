package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/beatlabs/patron/trace"
	"github.com/go-redis/redis/v7"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	versionTag = "version"
)

var (
	version = "dev"
)

// RedisSpan starts a new Redis child span with specified tags.
func RedisSpan(ctx context.Context, opName, cmp, dbType, instance, stmt string,
	tags ...opentracing.Tag) (opentracing.Span, context.Context) {

	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, cmp)
	ext.DBType.Set(sp, dbType)
	ext.DBInstance.Set(sp, instance)
	ext.DBStatement.Set(sp, stmt)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	sp.SetTag(versionTag, version)
	return sp, ctx
}

// Options wraps redis.Options for easier usage.
type Options redis.Options

// Empty represents the error which is returned in case a key is not found.
const Empty = redis.Nil

// Client represents a connection with a Redis client.
type Client struct {
	*redis.Client
}

func (c *Client) startSpan(ctx context.Context, opName, stmt string) (opentracing.Span, context.Context) {
	return RedisSpan(ctx, opName, trace.RedisComponent, trace.RedisDBType, stmt, c.Options().Addr)
}

// New returns a new Redis client.
func New(ctx context.Context, opt Options) *Client {
	clientOptions := redis.Options(opt)
	return &Client{redis.NewClient(&clientOptions)}
}

// Do creates and processes a custom Cmd on the underlying Redis client.
func (c *Client) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	sp, _ := c.startSpan(ctx, "redis.Do", fmt.Sprintf("%v", args))
	cmd := c.Client.Do(args...)
	trace.SpanComplete(sp, cmd.Err())
	return cmd.Result()
}

// Close closes the connection to the underlying Redis client.
func (c *Client) Close(ctx context.Context, args ...interface{}) error {
	sp, _ := c.startSpan(ctx, "redis.Close", "")
	cmd := c.Client.Close()
	trace.SpanComplete(sp, cmd)
	return errors.New(cmd.Error())
}

// Ping can be used to test whether a connection is still alive, or measure latency.
func (c *Client) Ping(ctx context.Context) (string, error) {
	sp, _ := c.startSpan(ctx, "redis.Ping", "")
	cmd := c.Client.Ping()
	trace.SpanComplete(sp, cmd.Err())
	return cmd.Result()
}
