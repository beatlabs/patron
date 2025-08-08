package es

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"go.opentelemetry.io/otel"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

// BenchmarkIndicesCreate measures the overhead of the ES client with OTEL tracing+metrics.
func BenchmarkIndicesCreate(b *testing.B) {
	b.ReportAllocs()

	// Use a manual metrics reader to avoid exporter costs during the benchmark.
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	otel.SetMeterProvider(provider)

	// Fake transport to avoid network I/O and isolate instrumentation overhead.
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		hdr := make(http.Header)
		hdr.Set("X-Elastic-Product", "Elasticsearch")
		return &http.Response{
			StatusCode: 200,
			Header:     hdr,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"acknowledged": true}`))),
			Request:    req,
		}, nil
	})

	cfg := elasticsearch.Config{Transport: rt, Addresses: []string{"http://es.local"}}
	client, err := New(cfg, "1.0.0")
	if err != nil {
		b.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	queryBody := `{"mappings": {"_doc": {"properties": {"field1": {"type": "integer"}}}}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsp, err := client.Indices.Create(
			"bench_index",
			client.Indices.Create.WithBody(strings.NewReader(queryBody)),
			client.Indices.Create.WithContext(ctx),
		)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		if rsp != nil && rsp.Body != nil {
			_ = rsp.Body.Close()
		}
	}
	b.StopTimer()

	// Shutdown provider outside of the measured section.
	_ = provider.Shutdown(ctx)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
