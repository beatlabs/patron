package cache

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// cacheHandlerRequest is the dedicated request object for the cache handler
type cacheHandlerRequest struct {
	header string
	path   string
	query  string
}

// toCacheHandlerRequest transforms the http Request object to the cache handler request
func toCacheHandlerRequest(req *http.Request) *cacheHandlerRequest {
	var header string
	if req.Header != nil {
		header = req.Header.Get(HeaderCacheControl)
	}
	var path string
	var query string
	if req.URL != nil {
		path = req.URL.Path
		query = req.URL.RawQuery
	}
	return &cacheHandlerRequest{
		header: header,
		path:   path,
		query:  query,
	}
}

// getKey generates a unique cache key based on the route path and the query parameters
func (c *cacheHandlerRequest) getKey() string {
	return fmt.Sprintf("%s:%s", c.path, c.query)
}

// CacheHandlerResponse is the dedicated Response object for the cache handler
type CacheHandlerResponse struct {
	Bytes  []byte
	Header http.Header
}

// CachedResponse is the struct representing an object retrieved or ready to be put into the route cache
type CachedResponse struct {
	Response  CacheHandlerResponse
	LastValid int64
	Etag      string
	Warning   string
	FromCache bool
	Err       error
}

func (c *CachedResponse) encode() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("could not encode cache response object: %w", err)
	}
	return b, nil
}

func (c *CachedResponse) decode(data []byte) error {
	return json.Unmarshal(data, c)
}
