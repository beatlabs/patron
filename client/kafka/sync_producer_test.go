package kafka

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncProducer_SendBatch_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty messages returns error", func(t *testing.T) {
		t.Parallel()

		sp := &SyncProducer{}
		ctx := context.Background()

		err := sp.SendBatch(ctx, []*sarama.ProducerMessage{})

		require.Error(t, err)
		assert.Equal(t, "messages are empty or nil", err.Error())
	})

	t.Run("nil messages returns error", func(t *testing.T) {
		t.Parallel()

		sp := &SyncProducer{}
		ctx := context.Background()

		err := sp.SendBatch(ctx, nil)

		require.Error(t, err)
		assert.Equal(t, "messages are empty or nil", err.Error())
	})
}

func TestStatusCountBatchAdd(t *testing.T) {
	t.Parallel()

	t.Run("processes empty batch", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		messages := []*sarama.ProducerMessage{}

		// Should not panic
		assert.NotPanics(t, func() {
			statusCountBatchAdd(ctx, topicAttribute("test"), messages)
		})
	})

	t.Run("processes single message", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		messages := []*sarama.ProducerMessage{
			{Topic: "test-topic"},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			statusCountBatchAdd(ctx, topicAttribute("test-topic"), messages)
		})
	})

	t.Run("processes multiple messages", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		messages := []*sarama.ProducerMessage{
			{Topic: "topic1"},
			{Topic: "topic2"},
			{Topic: "topic3"},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			statusCountBatchAdd(ctx, topicAttribute("batch"), messages)
		})
	})

	t.Run("handles cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		messages := []*sarama.ProducerMessage{
			{Topic: "test-topic"},
		}

		// Should not panic even with cancelled context
		assert.NotPanics(t, func() {
			statusCountBatchAdd(ctx, topicAttribute("test"), messages)
		})
	})
}

func TestDeliveryTypeConstants(t *testing.T) {
	t.Parallel()

	t.Run("sync delivery type", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "sync", deliveryTypeSync)
		assert.NotNil(t, deliveryTypeSyncAttr)
		assert.Equal(t, "delivery", string(deliveryTypeSyncAttr.Key))
		assert.Equal(t, "sync", deliveryTypeSyncAttr.Value.AsString())
	})
}

func TestSyncProducer_Structure(t *testing.T) {
	t.Parallel()

	t.Run("embeds base producer", func(t *testing.T) {
		t.Parallel()

		sp := &SyncProducer{
			baseProducer: baseProducer{},
		}

		assert.NotNil(t, sp)
		assert.NotNil(t, sp.baseProducer)
	})
}
