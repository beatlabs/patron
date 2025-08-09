package es

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNew(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	ctx := context.Background()

	shutdownProvider, collectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	responseMsg := `[{"acknowledged": true, "shards_acknowledged": true, "index": "test"}]`
	ctx, indexName := context.Background(), "test_index"
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("X-Elastic-Product", "Elasticsearch")
		_, err := w.Write([]byte(responseMsg))
		assert.NoError(t, err)
	}))
	listenCfg := &net.ListenConfig{}
	listener, err := listenCfg.Listen(context.Background(), "tcp", ":9200") //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	ts.Listener = listener
	ts.Start()
	defer ts.Close()

	host := "http://localhost:9200"

	cfg := elasticsearch.Config{
		Addresses: []string{host},
	}

	version := "1.0.0"
	client, err := New(cfg, version)
	require.NoError(t, err)
	assert.NotNil(t, client)

	queryBody := `{"mappings": {"_doc": {"properties": {"field1": {"type": "integer"}}}}}`

	rsp, err := client.Indices.Create(
		indexName,
		client.Indices.Create.WithBody(strings.NewReader(queryBody)),
		client.Indices.Create.WithContext(ctx),
	)
	require.NoError(t, err)
	assert.NotNil(t, rsp)

	// Traces
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))
	assert.Len(t, exp.GetSpans(), 1)

	// Metrics: ensure our ES histogram is emitted
	rm := collectMetrics(1)
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "es.request.duration" {
				found = true
			}
		}
	}
	assert.True(t, found, "expected es.request.duration metric to be recorded")
}

func TestNew_FailureMetrics(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	ctx := context.Background()

	shutdownProvider, collectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	// ES test server returning 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": true}`))
	}))
	defer ts.Close()

	cfg := elasticsearch.Config{
		Addresses: []string{ts.URL},
	}

	client, err := New(cfg, "1.0.0")
	require.NoError(t, err)

	// Make a typed API request (uses instrumentation Start/Close) that will return 500
	queryBody := `{"mappings": {"_doc": {"properties": {"field1": {"type": "integer"}}}}}`
	rsp, err := client.Indices.Create(
		"failed_index",
		client.Indices.Create.WithBody(strings.NewReader(queryBody)),
		client.Indices.Create.WithContext(ctx),
	)
	require.NoError(t, err)
	assert.NotNil(t, rsp)
	_ = rsp.Body.Close()

	// Traces
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	// Metrics: ensure our ES histogram is emitted even on failure
	rm := collectMetrics(1)
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "es.request.duration" {
				found = true
			}
		}
	}
	assert.True(t, found, "expected es.request.duration metric to be recorded on failure")
}
