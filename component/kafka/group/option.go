package group

import (
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"golang.org/x/exp/slog"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// WithFailureStrategy sets the strategy to follow for the component when it encounters an error.
// The kafka.ExitStrategy will fail the component, if there are retries > 0 then the component will reconnect and retry
// the failed message.
// The kafka.SkipStrategy will skip the message on failure. If a client wants to retry a message before failing then
// this needs to be handled in the kafka.BatchProcessorFunc.
func WithFailureStrategy(fs kafka.FailStrategy) OptionFunc {
	return func(c *Component) error {
		if fs > kafka.SkipStrategy || fs < kafka.ExitStrategy {
			return errors.New("invalid failure strategy provided")
		}
		c.failStrategy = fs
		return nil
	}
}

// WithCheckTopic checks whether the component-configured topics exist in the broker.
func WithCheckTopic() OptionFunc {
	return func(c *Component) error {
		saramaConf := sarama.NewConfig()
		client, err := sarama.NewClient(c.brokers, saramaConf)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer func() { _ = client.Close() }()
		brokerTopics, err := client.Topics()
		if err != nil {
			return fmt.Errorf("failed to get topics from broker: %w", err)
		}

		topicsSet := make(map[string]struct{}, len(brokerTopics))
		for _, topic := range brokerTopics {
			topicsSet[topic] = struct{}{}
		}

		for _, topic := range c.topics {
			if _, ok := topicsSet[topic]; !ok {
				return fmt.Errorf("topic %s does not exist in broker", topic)
			}
		}
		return nil
	}
}

// WithRetries sets the number of time a component should retry in case of an error.
// These retries are depleted in these cases:
// * when there are temporary connection issues
// * a message batch fails to be processed through the user-defined processing function and the failure strategy is set to kafka.ExitStrategy
// * any other reason for which the component needs to reconnect.
func WithRetries(count uint) OptionFunc {
	return func(c *Component) error {
		c.retries = count
		return nil
	}
}

// WithRetryWait sets the wait period for the component retry.
func WithRetryWait(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("retry wait time should be a positive number")
		}
		c.retryWait = interval
		return nil
	}
}

// WithBatchSize sets the message batch size the component should process at once.
func WithBatchSize(size uint) OptionFunc {
	return func(c *Component) error {
		if size == 0 {
			return errors.New("zero batch size provided")
		}
		c.batchSize = size
		return nil
	}
}

// WithBatchTimeout sets the message batch timeout. If the desired batch size is not reached and if the timeout elapses
// without new messages coming in, the messages in the buffer would get processed as a batch.
func WithBatchTimeout(timeout time.Duration) OptionFunc {
	return func(c *Component) error {
		if timeout < 0 {
			return errors.New("batch timeout should greater than or equal to zero")
		}
		c.batchTimeout = timeout
		return nil
	}
}

// WithBatchMessageDeduplication enables the deduplication of messages based on the message's key.
// This implementation does not do additional sorting, but instead relies on the ordering guarantees that Kafka gives
// within partitions of a topic. Don't use this functionality if you've changed your producer's partition hashing
// behaviour to a nondeterministic way.
func WithBatchMessageDeduplication() OptionFunc {
	return func(c *Component) error {
		c.batchMessageDeduplication = true
		return nil
	}
}

// WithCommitSync instructs the consumer to commit offsets in a blocking operation after processing every batch of messages.
func WithCommitSync() OptionFunc {
	return func(c *Component) error {
		if c.saramaConfig != nil && c.saramaConfig.Consumer.Offsets.AutoCommit.Enable {
			// redundant commits warning
			slog.Warn("consumer is set to commit offsets after processing each batch and auto-commit is enabled")
		}
		c.commitSync = true
		return nil
	}
}

// WithNewSessionCallback adds a callback when a new consumer group session is created (e.g., rebalancing).
func WithNewSessionCallback(sessionCallback func(sarama.ConsumerGroupSession) error) OptionFunc {
	return func(c *Component) error {
		if sessionCallback == nil {
			return errors.New("nil session callback")
		}

		c.sessionCallback = sessionCallback
		return nil
	}
}
