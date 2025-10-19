// Package es provides a client with included tracing capabilities.
package es

import (
	"github.com/elastic/go-elasticsearch/v8"
)

// New creates a new elasticsearch client with tracing capabilities.
func New(cfg elasticsearch.Config, version string) (*elasticsearch.Client, error) {
	// Enable tracing via upstream OTEL instrumentation and record metrics via our wrapper
	instr, err := newMetricInstrumentation(version)
	if err != nil {
		return nil, err
	}
	cfg.Instrumentation = instr

	return elasticsearch.NewClient(cfg)
}
