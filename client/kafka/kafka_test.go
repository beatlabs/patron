package kafka

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Create(t *testing.T) {
	t.Parallel()
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value\n"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified\n"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.brokers, tt.args.cfg).Create()

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
		})
	}
}

func TestBuilder_CreateAsync(t *testing.T) {
	t.Parallel()
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value\n"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified\n"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, chErr, err := New(tt.args.brokers, tt.args.cfg).CreateAsync()

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
			require.Nil(t, chErr)
		})
	}
}

func TestDefaultProducerSaramaConfig(t *testing.T) {
	sc, err := DefaultProducerSaramaConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, fmt.Sprintf("-%s", "name")))
	require.True(t, sc.Producer.Idempotent)

	sc, err = DefaultProducerSaramaConfig("name", false)
	require.NoError(t, err)
	require.False(t, sc.Producer.Idempotent)
}

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
