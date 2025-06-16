package v2

import (
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	sc, err := DefaultConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, "-name"))
	require.True(t, sc.Producer.Idempotent)

	sc, err = DefaultConfig("name", false)
	require.NoError(t, err)
	require.False(t, sc.Producer.Idempotent)
}

func TestNewSyncProducer(t *testing.T) {
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
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no sarama configuration specified"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := NewSyncProducer(tt.args.brokers, tt.args.cfg)

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
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no sarama configuration specified"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := NewAsyncProducer(tt.args.brokers, tt.args.cfg)

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
		})
	}
}
