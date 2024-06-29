// Package redis provides a client with included tracing capabilities.
package redis

import (
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// New returns a new Redis client.
func New(opt *redis.Options) (*redis.Client, error) {
	cl := redis.NewClient(opt)

	if err := redisotel.InstrumentTracing(cl); err != nil {
		return nil, err
	}
	if err := redisotel.InstrumentMetrics(cl); err != nil {
		return nil, err
	}

	return cl, nil
}
