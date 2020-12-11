// Package redis provides a client with included tracing capabilities.
package redis

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/trace"
	"github.com/go-redis/redis/v7"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	component = "redis"
	dbType    = "In-memory"
)

// Options wraps redis.Options for easier usage.
type Options redis.Options

// Nil represents the error which is returned in case a key is not found.
const Nil = redis.Nil

// Client represents a connection with a Redis client.
type Client struct {
	*redis.Client
}

func (c *Client) startSpan(ctx context.Context, opName, stmt string, tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, component)
	ext.DBType.Set(sp, dbType)
	ext.DBInstance.Set(sp, c.Options().Addr)
	ext.DBStatement.Set(sp, stmt)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	return sp, ctx
}

// New returns a new Redis client.
func New(opt Options) *Client {
	clientOptions := redis.Options(opt)
	return &Client{redis.NewClient(&clientOptions)}
}

// Do creates and processes a custom Cmd on the underlying Redis client.
func (c *Client) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	sp, _ := c.startSpan(ctx, "redis.Do", fmt.Sprintf("%v", args))
	cmd := c.Client.Do(args...)
	trace.SpanComplete(sp, cmd.Err())
	return cmd
}

// Close closes the connection to the underlying Redis client.
func (c *Client) Close(ctx context.Context, _ ...interface{}) error {
	sp, _ := c.startSpan(ctx, "redis.Close", "")
	err := c.Client.Close()
	trace.SpanComplete(sp, err)
	return err
}

// Ping contacts the redis client, and returns 'PONG' if the client is reachable.
// It can be used to test whether a connection is still alive, or measure latency.
func (c *Client) Ping(ctx context.Context) *redis.StatusCmd {
	sp, _ := c.startSpan(ctx, "redis.Ping", "")
	cmd := c.Client.Ping()
	trace.SpanComplete(sp, cmd.Err())
	return cmd
}
