// Package cache provides abstractions to allow the creation of concrete implementations.
package cache

import (
	"context"
	"time"
)

// Cache interface that defines the methods required.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool, error)
	Purge(ctx context.Context) error
	Remove(ctx context.Context, key string) error
	Set(ctx context.Context, key string, value interface{}) error
}

// TTLCache interface adds support for expiring key value pairs.
type TTLCache interface {
	Cache
	SetTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
