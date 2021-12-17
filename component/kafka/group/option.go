package group

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/log"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// FailureStrategy sets the strategy to follow for the component when it encounters an error.
// The kafka.ExitStrategy will fail the component, if there are Retries > 0 then the component will reconnect and retry
// the failed message.
// The kafka.SkipStrategy will skip the message on failure. If a client wants to retry a message before failing then
// this needs to be handled in the kafka.BatchProcessorFunc.
func FailureStrategy(fs kafka.FailStrategy) OptionFunc {
	return func(c *Component) error {
		if fs > kafka.SkipStrategy || fs < kafka.ExitStrategy {
			return errors.New("invalid failure strategy provided")
		}
		c.failStrategy = fs
		return nil
	}
}

// CheckTopic will attempt to:
//	1. connect to the broker,
//  2. retrieve the existing topics in the broker
//  3. if auto-topic creation is disabled, check whether the configured topics exist in the broker
// If any of the checks above fail the component will exit before starting to consume messages
func CheckTopic() OptionFunc {
	return func(c *Component) error {
		c.topicChecker = func(ctx context.Context, c *Component) error {
			client, err := sarama.NewClient(c.brokers, c.saramaConfig)
			defer func() {
				if client != nil {
					_ = client.Close()
				}
			}()
			// without client no connection can be established
			if err != nil {
				return err
			}
			// check if the topic exists
			brokerTopics, err := client.Topics()
			if err != nil {
				return err
			}

			// if auto-topic creation is not enabled then check if the topic exists
			if !c.saramaConfig.Metadata.AllowAutoTopicCreation {
				for _, topic := range c.topics {
					topicExists := false
					for _, brokerTopic := range brokerTopics {
						if topic == brokerTopic {
							topicExists = true
							break
						}
					}
					if !topicExists {
						return fmt.Errorf("topic %s does not exist in broker", topic)
					}
				}
			}

			return nil
		}
		return nil
	}
}

// Retries sets the number of time a component should retry in case of an error.
// These retries are depleted in these cases:
// * when there are temporary connection issues
// * a message batch fails to be processed through the user-defined processing function and the failure strategy is set to kafka.ExitStrategy
// * any other reason for which the component needs to reconnect.
func Retries(count uint) OptionFunc {
	return func(c *Component) error {
		c.retries = count
		return nil
	}
}

// RetryWait sets the wait period for the component retry.
func RetryWait(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("retry wait time should be a positive number")
		}
		c.retryWait = interval
		return nil
	}
}

// BatchSize sets the message batch size the component should process at once.
func BatchSize(size uint) OptionFunc {
	return func(c *Component) error {
		if size == 0 {
			return errors.New("zero batch size provided")
		}
		c.batchSize = size
		return nil
	}
}

// BatchTimeout sets the message batch timeout. If the desired batch size is not reached and if the timeout elapses
// without new messages coming in, the messages in the buffer would get processed as a batch.
func BatchTimeout(timeout time.Duration) OptionFunc {
	return func(c *Component) error {
		if timeout < 0 {
			return errors.New("batch timeout should greater than or equal to zero")
		}
		c.batchTimeout = timeout
		return nil
	}
}

// CommitSync instructs the consumer to commit offsets in a blocking operation after processing every batch of messages
func CommitSync() OptionFunc {
	return func(c *Component) error {
		if c.saramaConfig != nil && c.saramaConfig.Consumer.Offsets.AutoCommit.Enable {
			// redundant commits warning
			log.Warn("consumer is set to commit offsets after processing each batch and auto-commit is enabled")
		}
		c.commitSync = true
		return nil
	}
}
