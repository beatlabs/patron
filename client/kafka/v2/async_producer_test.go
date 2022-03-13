package v2

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func Test_injectTracingAndCorrelationHeaders(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })
	ctx := correlation.ContextWithID(context.Background(), "123")
	sp, _ := trace.ChildSpan(context.Background(), trace.ComponentOpName(componentTypeAsync, "topic"), componentTypeAsync,
		ext.SpanKindProducer, asyncTag, opentracing.Tag{Key: "topic", Value: "topic"})
	msg := sarama.ProducerMessage{}
	assert.NoError(t, injectTracingAndCorrelationHeaders(ctx, &msg, sp))
	assert.Len(t, msg.Headers, 4)
	assert.Equal(t, correlation.HeaderID, string(msg.Headers[0].Key))
	assert.Equal(t, "123", string(msg.Headers[0].Value))
	assert.Equal(t, "mockpfx-ids-traceid", string(msg.Headers[1].Key))
	assert.Equal(t, "mockpfx-ids-spanid", string(msg.Headers[2].Key))
	assert.Equal(t, "mockpfx-ids-sampled", string(msg.Headers[3].Key))
}
