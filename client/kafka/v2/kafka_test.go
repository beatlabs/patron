package v2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_Create(t *testing.T) {
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value\n"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "config is nil\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.brokers).WithConfig(tt.args.cfg).Create()

			assert.EqualError(t, err, tt.expectedErr)
			assert.Nil(t, got)
		})
	}
}

func TestBuilder_CreateAsync(t *testing.T) {
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value\n"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "config is nil\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, chErr, err := New(tt.args.brokers).WithConfig(tt.args.cfg).CreateAsync()

			assert.EqualError(t, err, tt.expectedErr)
			assert.Nil(t, got)
			assert.Nil(t, chErr)
		})
	}
}

func Test_DefaultConsumerSaramaConfig(t *testing.T) {
	sc, err := DefaultConsumerSaramaConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, fmt.Sprintf("-%s", "name")))
	require.Equal(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)

	sc, err = DefaultConsumerSaramaConfig("name", false)
	require.NoError(t, err)
	require.NotEqual(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)
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
