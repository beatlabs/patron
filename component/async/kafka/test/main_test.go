// +build integration

package simple

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	dockerKafka "github.com/beatlabs/patron/test/docker/kafka"
)

const (
	topic = "Topic"
)

func TestMain(m *testing.M) {
	os.Exit(dockerKafka.RunWithKafka(m, 120*time.Second, getTopic(topic)))
}

func getTopic(name string) string {
	return fmt.Sprintf("%s:1:1", name)
}

func getProducerMessage(message string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}
}

func consumeMessages(consumer async.Consumer, expectedMessageCount int) ([]string, error) {

	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	ch, chErr, err := consumer.Consume(ctx)
	if err != nil {
		return nil, err
	}

	received := make([]string, 0, expectedMessageCount)

	for {
		select {
		case msg := <-ch:
			received = append(received, string(msg.Payload()))
			expectedMessageCount--
			if expectedMessageCount == 0 {
				return received, nil
			}
		case err := <-chErr:
			return nil, err
		}
	}
}
