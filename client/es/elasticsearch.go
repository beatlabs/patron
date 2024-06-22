// Package es provides a client with included tracing capabilities.
package es

import (
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
)

// TODO: Metrics are not supported yet.

// New creates a new elasticsearch client with tracing capabilities.
func New(cfg elasticsearch.Config, version string) (*elasticsearch.Client, error) {
	cfg.Instrumentation = elastictransport.NewOtelInstrumentation(nil, false, version)

	return elasticsearch.NewClient(cfg)
}
