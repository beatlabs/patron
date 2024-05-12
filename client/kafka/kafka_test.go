package kafka

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified"},
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
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified"},
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
	exp := tracetest.NewInMemoryExporter()
	_, err := patrontrace.Setup("test", nil, exp)
	require.NoError(t, err)

	ctx := correlation.ContextWithID(context.Background(), "123")

	msg := sarama.ProducerMessage{}

	ctx, _ = startSpan(ctx, "send", deliveryTypeSync, "topic")

	injectTracingAndCorrelationHeaders(ctx, &msg)
	assert.Len(t, msg.Headers, 2)
	assert.Equal(t, correlation.HeaderID, string(msg.Headers[0].Key))
	assert.Equal(t, "123", string(msg.Headers[0].Value))
	assert.Equal(t, "traceparent", string(msg.Headers[1].Key))
	assert.NotEmpty(t, string(msg.Headers[1].Value))
}
