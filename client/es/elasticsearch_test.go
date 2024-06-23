package es

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beatlabs/patron/observability/trace"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNew(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	// Setup metrics
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetMeterProvider(provider)

	responseMsg := `[{"acknowledged": true, "shards_acknowledged": true, "index": "test"}]`
	ctx, indexName := context.Background(), "test_index"
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("X-Elastic-Product", "Elasticsearch")
		_, err := w.Write([]byte(responseMsg))
		assert.NoError(t, err)
	}))
	listener, err := net.Listen("tcp", ":9200") //nolint:gosec
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
	assert.NoError(t, err)
	assert.NotNil(t, client)

	queryBody := `{"mappings": {"_doc": {"properties": {"field1": {"type": "integer"}}}}}`

	rsp, err := client.Indices.Create(
		indexName,
		client.Indices.Create.WithBody(strings.NewReader(queryBody)),
		client.Indices.Create.WithContext(ctx),
	)
	assert.NoError(t, err)
	assert.NotNil(t, rsp)

	// Traces
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))
	assert.Len(t, exp.GetSpans(), 1)
}
