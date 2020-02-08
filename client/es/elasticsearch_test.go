package es

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/beatlabs/patron/trace"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/estransport"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestStartEndSpan(t *testing.T) {
	defaultAddr := getAddrFromEnv()
	hosts := []string{defaultAddr, "http://10.1.1.1:9200"}
	body, user, method := `{"field1": "10"}`, "user1", "PUT"

	req, err := http.NewRequest(method, defaultAddr, strings.NewReader(body))
	assert.NoError(t, err)

	tracingInfo := tracingInfo{
		user:  user,
		hosts: hosts,
	}
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)

	sp, err := tracingInfo.startSpan(req)
	assert.NoError(t, err)
	assert.NotNil(t, sp)
	assert.Empty(t, mtr.FinishedSpans())
	assert.IsType(t, &mocktracer.MockSpan{}, sp)

	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	actualTags := jsp.Tags()

	assert.Equal(t, body, actualTags["db.statement"])

	hostsFmt := "[" + strings.Join(hosts, ", ") + "]"
	assert.EqualValues(t, hostsFmt, actualTags["hosts"])

	assert.Equal(t, "dev", actualTags["version"])
	assert.Equal(t, "go-elasticsearch", actualTags["component"])
	assert.Equal(t, "elasticsearch", actualTags["db.type"])
	assert.Equal(t, user, actualTags["db.user"])
	assert.Equal(t, "/", actualTags["http.url"])
	assert.Equal(t, method, actualTags["http.method"])

	respondent := "es.respondent.com:9200"
	statusCode := 200
	rsp := &http.Response{
		Request: &http.Request{
			URL: &url.URL{
				Host: respondent,
			},
		},
		StatusCode: statusCode,
	}
	endSpan(sp, rsp)

	jsp = sp.(*mocktracer.MockSpan)
	assert.Equal(t, respondent, jsp.Tag(respondentTag))
	assert.Equal(t, uint16(statusCode), jsp.Tag("http.status_code"))
	assert.Equal(t, false, jsp.Tag("error"))

	actualResponseTags := jsp.Tags()
	delete(actualResponseTags, "http.status_code")
	delete(actualResponseTags, respondentTag)
	delete(actualResponseTags, "error")
	assert.EqualValues(t, actualTags, actualResponseTags)
}

func TestNewDefaultClient(t *testing.T) {
	newClient, err := NewDefaultClient()
	assert.NoError(t, err)

	upstreamClient, err := elasticsearch.NewDefaultClient()
	assert.NoError(t, err)
	assert.IsType(t, *upstreamClient, newClient.Client)

	expectedTransport, transport := new(transportClient), newClient.Transport
	assert.IsType(t, expectedTransport, transport)

	defaultAddr := getAddrFromEnv()
	expectedURL, err := url.Parse(strings.TrimRight(defaultAddr, "/"))
	assert.NoError(t, err)
	cfg := estransport.Config{
		URLs:      []*url.URL{expectedURL},
		Transport: nil,
	}
	expectedTransport.client = estransport.New(cfg)
	expectedTransport.tracingInfo.hosts = []string{defaultAddr}
	assert.EqualValues(t, expectedTransport, transport)
}

func TestNewClient(t *testing.T) {
	addresses := []string{"http://www.host1.com:9200", "https://10.1.1.1:9300"}
	user, password, apiKey := "user1", "pass", "key"
	cfg := Config{
		Addresses: addresses,
		Username:  user,
		Password:  password,
		APIKey:    apiKey,
	}

	newClient, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.IsType(t, new(Client), newClient)

	expectedTransport, transport := new(transportClient), newClient.Transport
	assert.IsType(t, expectedTransport, transport)

	expectedURLs, err := addrsToURLs(addresses)
	assert.NoError(t, err)
	transportCfg := estransport.Config{
		URLs:      expectedURLs,
		Username:  user,
		Password:  password,
		APIKey:    apiKey,
		Transport: nil,
		Logger:    nil,
	}
	expectedTransport.client = estransport.New(transportCfg)
	expectedTransport.tracingInfo.hosts = addresses
	expectedTransport.user = user
	assert.EqualValues(t, expectedTransport, transport)
}

func TestEsQuery(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	assert.Empty(t, mtr.FinishedSpans())

	responseMsg := `[{"acknowledged": true, "shards_acknowledged": true, "index": "test"}]`
	ctx, indexName := context.Background(), "test_index"
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(responseMsg))
		assert.NoError(t, err)
	}))
	listener, err := net.Listen("tcp", ":"+defaultPort)
	if err != nil {
		t.Fatal(err)
	}
	ts.Listener = listener
	ts.Start()
	defer ts.Close()

	queryBody := `{"mappings": {"_doc": {"properties": {"field1": {"type": "integer"}}}}}`
	esClient, err := NewDefaultClient()
	assert.NoError(t, err)
	rsp, err := esClient.Indices.Create(
		indexName,
		esClient.Indices.Create.WithBody(strings.NewReader(queryBody)),
		esClient.Indices.Create.WithContext(ctx),
	)
	assert.NoError(t, err)
	assert.NotNil(t, rsp)

	finishedSpans := mtr.FinishedSpans()
	assert.Equal(t, 1, len(finishedSpans))

	assert.Equal(t, opName, finishedSpans[0].OperationName)
}

func TestGetAddrFromEnv(t *testing.T) {
	addr := getAddrFromEnv()
	assert.Equal(t, defaultHost+":"+defaultPort, addr)

	err := os.Setenv(defaultHostEnv, "http://10.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv(defaultPortEnv, "9300")
	if err != nil {
		t.Fatal(err)
	}
	addr = getAddrFromEnv()
	assert.Equal(t, "http://10.1.1.1:9300", addr)
}

func TestStartSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)

	hostPool := []string{"http://localhost:9200", "http:10.1.1.1:9201", "https://www.domain.com:9203"}
	tracingInfo := tracingInfo{
		user:  "es-user",
		hosts: hostPool,
	}
	req, err := http.NewRequest("query-method", "es-uri", strings.NewReader("query-body"))
	assert.NoError(t, err)

	sp, err := tracingInfo.startSpan(req)
	assert.NoError(t, err)
	assert.NotNil(t, sp)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	trace.SpanSuccess(sp)
	rawspan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component":    "go-elasticsearch",
		"version":      "dev",
		"db.statement": "query-body",
		"db.type":      "elasticsearch",
		"db.user":      "es-user",
		"http.url":     "es-uri",
		"http.method":  "query-method",
		trace.HostsTag: "[http://localhost:9200, http:10.1.1.1:9201, https://www.domain.com:9203]",
		"error":        false,
	}, rawspan.Tags())
}
