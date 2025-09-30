package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeliveryTypeAsync(t *testing.T) {
	t.Parallel()

	t.Run("async delivery type constant", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "async", deliveryTypeAsync)
	})

	t.Run("async delivery type attribute", func(t *testing.T) {
		t.Parallel()

		assert.NotNil(t, deliveryTypeAsyncAttr)
		assert.Equal(t, "delivery", string(deliveryTypeAsyncAttr.Key))
		assert.Equal(t, "async", deliveryTypeAsyncAttr.Value.AsString())
	})
}

func TestAsyncProducer_Structure(t *testing.T) {
	t.Parallel()

	t.Run("embeds base producer", func(t *testing.T) {
		t.Parallel()

		ap := &AsyncProducer{
			baseProducer: baseProducer{},
		}

		assert.NotNil(t, ap)
		assert.NotNil(t, ap.baseProducer)
	})
}
