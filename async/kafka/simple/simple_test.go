package simple

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestConsumer_ConsumeWithoutGroup(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	topic := "foo_topic"
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetVersion(1).
			SetOffset(topic, 0, sarama.OffsetNewest, 10).
			SetOffset(topic, 0, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(topic, 0, 9, sarama.StringEncoder("Foo")),
	})

	f, err := New("name", topic, []string{broker.Addr()})
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	ctx := context.Background()
	chMsg, chErr, err := c.Consume(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, chMsg)
	assert.NotNil(t, chErr)

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()

	ctx.Done()
}
