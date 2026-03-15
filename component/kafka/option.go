package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// WithFailureStrategy sets the strategy to follow for the component when it encounters an error.
// The ExitStrategy will fail the component, if there are retries > 0 then the component will reconnect and retry
// the failed message.
// The SkipStrategy will skip the message on failure. If a client wants to retry a message before failing then
// this needs to be handled in the BatchProcessorFunc.
func WithFailureStrategy(fs FailStrategy) OptionFunc {
	return func(c *Component) error {
		if fs > SkipStrategy || fs < ExitStrategy {
			return errors.New("invalid failure strategy provided")
		}
		c.failStrategy = fs
		return nil
	}
}

// WithCheckTopic checks whether the component-configured topics exist in the broker.
func WithCheckTopic() OptionFunc {
	return func(c *Component) error {
		cl, err := kgo.NewClient(kgo.SeedBrokers(c.brokers...))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer cl.Close()

		req := kmsg.NewPtrMetadataRequest()
		for _, topic := range c.topics {
			t := topic
			rt := kmsg.NewMetadataRequestTopic()
			rt.Topic = &t
			req.Topics = append(req.Topics, rt)
		}

		shards := cl.RequestSharded(context.Background(), req)
		existingTopics := make(map[string]struct{}, len(c.topics))
		for _, shard := range shards {
			if shard.Err != nil {
				return fmt.Errorf("failed to get topics from broker: %w", shard.Err)
			}

			resp, ok := shard.Resp.(*kmsg.MetadataResponse)
			if !ok {
				continue
			}

			for _, topic := range resp.Topics {
				if topic.Topic != nil && topic.ErrorCode == 0 {
					existingTopics[*topic.Topic] = struct{}{}
				}
			}
		}

		for _, topic := range c.topics {
			if _, ok := existingTopics[topic]; !ok {
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
func WithRetries(count uint32) OptionFunc {
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

// WithManualCommit instructs the consumer to disable autocommit, block rebalances during processing,
// and commit offsets explicitly after each successfully processed batch.
func WithManualCommit() OptionFunc {
	return func(c *Component) error {
		c.manualCommit = true
		return nil
	}
}

// WithNewSessionCallback adds a callback when a new consumer group session is created (e.g., rebalancing).
func WithNewSessionCallback(sessionCallback func() error) OptionFunc {
	return func(c *Component) error {
		if sessionCallback == nil {
			return errors.New("nil session callback")
		}

		c.sessionCallback = sessionCallback
		return nil
	}
}
