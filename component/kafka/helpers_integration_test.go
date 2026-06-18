//go:build integration

package kafka

import (
	"testing"

	"github.com/beatlabs/patron/correlation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKafkaHelpersIntegration(t *testing.T) {
	t.Run("correlation id", func(t *testing.T) {
		assert.Equal(t, "known", getCorrelationID([]kgo.RecordHeader{{Key: correlation.HeaderID, Value: []byte("known")}}))

		got := getCorrelationID([]kgo.RecordHeader{{Key: correlation.HeaderID}})
		_, err := uuid.Parse(got)
		require.NoError(t, err)

		got = getCorrelationID(nil)
		_, err = uuid.Parse(got)
		require.NoError(t, err)
	})

	t.Run("deduplicate records keeps latest in original order", func(t *testing.T) {
		records := []*kgo.Record{
			{Key: []byte("a"), Value: []byte("a1")},
			{Key: []byte("b"), Value: []byte("b1")},
			{Key: []byte("a"), Value: []byte("a2")},
		}

		got := deduplicateRecords(records)

		require.Len(t, got, 2)
		assert.Equal(t, []byte("b1"), got[0].Value)
		assert.Equal(t, []byte("a2"), got[1].Value)
	})
}
