package lru

import (
	lru "github.com/hashicorp/golang-lru"
)

// Cache encapsulates a thread-safe fixed size LRU cache
// as defined in hashicorp/golang-lru.
type Cache struct {
	cache *lru.Cache
}

// New returns a new LRU cache that can hold 'size' number of keys at a time.
func New(size int) (*Cache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &Cache{cache: cache}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	value, ok := c.cache.Get(key)
	return value, ok, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() error {
	c.cache.Purge()
	return nil
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) error {
	c.cache.Remove(key)
	return nil
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	c.cache.Add(key, value)
	return nil
}
