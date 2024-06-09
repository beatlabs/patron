// Package cache provides abstractions to allow the creation of concrete implementations.
package cache

import (
	"context"
	"time"
)

// Cache interface that defines the methods required.
type Cache interface {
	// Get a value based on a specific key. The call returns whether the  key exists or not.
	Get(ctx context.Context, key string) (interface{}, bool, error)
	// Purge the cache.
	Purge(ctx context.Context) error
	// Remove the key from the cache.
	Remove(ctx context.Context, key string) error
	// Set the value for the specified key.
	Set(ctx context.Context, key string, value interface{}) error
}

// TTLCache interface adds support for expiring key value pairs.
type TTLCache interface {
	Cache
	// SetTTL sets the value of a specified key with a time to live.
	SetTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
