// Package cache provide abstractions for concrete cache implementations.
package cache

import (
	"time"
)

// Cache interface.
type Cache interface {
	Get(key string) (interface{}, bool, error)
	Purge() error
	Remove(key string) error
	Set(key string, value interface{}) error
}

// TTLCache interface adds support for expiring key-value pairs.
type TTLCache interface {
	Cache
	SetTTL(key string, value interface{}, ttl time.Duration) error
}
