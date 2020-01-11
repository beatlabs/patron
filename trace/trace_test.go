package trace

import (
	"context"
	"net/http"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestSetup_Tracer_Close(t *testing.T) {
	err := Setup("TEST", "1.0.0", "0.0.0.0:6831", "const", 1)
	assert.NoError(t, err)
	err = Close()
	assert.NoError(t, err)
	version = "dev"
}

func TestStartFinishConsumerSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	hdr := map[string]string{"key": "val"}
	sp, ctx := ConsumerSpan(context.Background(), "123", AMQPConsumerComponent, "corID", hdr)
	assert.NotNil(t, sp)
	assert.NotNil(t, ctx)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "123", jsp.OperationName)
	SpanError(sp)
	assert.NotNil(t, sp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"span.kind":     ext.SpanKindConsumerEnum,
		"component":     "amqp-consumer",
		"error":         true,
		"version":       "dev",
		"correlationID": "corID",
	}, rawSpan.Tags())
}

func TestStartFinishChildSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	tag := opentracing.Tag{Key: "key", Value: "value"}
	sp, ctx := ConsumerSpan(context.Background(), "123", AMQPConsumerComponent, "corID", nil, tag)
	assert.NotNil(t, sp)
	assert.NotNil(t, ctx)
	childSp, childCtx := ChildSpan(ctx, "123", "cmp", tag)
	assert.NotNil(t, childSp)
	assert.NotNil(t, childCtx)
	childSp.LogKV("log event")
	assert.IsType(t, &mocktracer.MockSpan{}, childSp)
	jsp := childSp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "123", jsp.OperationName)
	SpanError(childSp)
	assert.NotNil(t, childSp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component": "cmp",
		"error":     true,
		"key":       "value",
		"version":   "dev",
	}, rawSpan.Tags())
	SpanSuccess(sp)
	rawSpan = mtr.FinishedSpans()[1]
	assert.Equal(t, map[string]interface{}{
		"component":     "amqp-consumer",
		"error":         false,
		"version":       "dev",
		"key":           "value",
		"span.kind":     ext.SpanKindConsumerEnum,
		"correlationID": "corID",
	}, rawSpan.Tags())
}

func TestHTTPStartFinishSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	sp, req := HTTPSpan("/", "corID", req)
	assert.NotNil(t, sp)
	assert.NotNil(t, req)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "GET /", jsp.OperationName)
	FinishHTTPSpan(jsp, 200)
	assert.NotNil(t, jsp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"span.kind":        ext.SpanKindRPCServerEnum,
		"component":        "http",
		"error":            false,
		"http.method":      "GET",
		"http.status_code": uint16(200),
		"http.url":         "/",
		"version":          "dev",
		"correlationID":    "corID",
	}, rawSpan.Tags())
}

func TestSQLStartFinishSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	tag := opentracing.Tag{Key: "key", Value: "value"}
	sp, req := SQLSpan(context.Background(), "name", "sql", "rdbms", "instance", "sa", "ssf", tag)
	assert.NotNil(t, sp)
	assert.NotNil(t, req)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	SpanSuccess(sp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component":    "sql",
		"version":      "dev",
		"db.instance":  "instance",
		"db.statement": "ssf",
		"db.type":      "rdbms",
		"db.user":      "sa",
		"error":        false,
		"key":          "value",
	}, rawSpan.Tags())
}

func TestEsSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	hostPool := []string{"http://localhost:9200", "http:10.1.1.1:9201", "https://www.domain.com:9203"}
	sp := EsSpan(context.Background(), "opName", "es-component", "es-user", "es-uri", "query-method", "query-body", hostPool)
	assert.NotNil(t, sp)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	SpanSuccess(sp)
	rawspan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component":    "es-component",
		"version":      "dev",
		"db.statement": "query-body",
		"db.type":      "elasticsearch",
		"db.user":      "es-user",
		"http.url":     "es-uri",
		"http.method":  "query-method",
		hostsTag:       "[http://localhost:9200, http:10.1.1.1:9201, https://www.domain.com:9203]",
		"error":        false,
	}, rawspan.Tags())
}

func TestComponentOpName(t *testing.T) {
	assert.Equal(t, "cmp target", ComponentOpName("cmp", "target"))
}
