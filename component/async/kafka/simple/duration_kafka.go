package simple

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
)

type outOfRangeOffsetError struct {
	message string
}

func (e *outOfRangeOffsetError) Error() string {
	return e.message
}

func (e *outOfRangeOffsetError) Is(target error) bool {
	_, ok := target.(*outOfRangeOffsetError) //nolint:errorlint
	return ok
}

type durationKafkaClientAPI interface {
	getPartitionIDs(topic string) ([]int32, error)
	getOldestOffset(topic string, partitionID int32) (int64, error)
	getNewestOffset(topic string, partitionID int32) (int64, error)
	getMessageAtOffset(ctx context.Context, topic string, partitionID int32, offset int64) (*sarama.ConsumerMessage, error)
}

type durationKafkaClient struct {
	kafkaClient     sarama.Client
	kafkaConsumer   sarama.Consumer
	consumerTimeout time.Duration
}

func newDurationKafkaClient(kafkaClient sarama.Client, kafkaConsumer sarama.Consumer, consumerTimeout time.Duration) (durationKafkaClient, error) {
	if kafkaClient == nil {
		return durationKafkaClient{}, errors.New("kafka client is nil")
	}
	if kafkaConsumer == nil {
		return durationKafkaClient{}, errors.New("kafka consumer is nil")
	}

	return durationKafkaClient{
		kafkaClient:     kafkaClient,
		kafkaConsumer:   kafkaConsumer,
		consumerTimeout: consumerTimeout,
	}, nil
}

func (c durationKafkaClient) getPartitionIDs(topic string) ([]int32, error) {
	partitionIDs, err := c.kafkaConsumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("unable to query Kafka to retrieve the partitions of the topic %s: %w", topic, err)
	}

	return partitionIDs, nil
}

func (c durationKafkaClient) getOldestOffset(topic string, partitionID int32) (int64, error) {
	offset, err := c.kafkaClient.GetOffset(topic, partitionID, sarama.OffsetOldest)
	if err != nil {
		return 0, fmt.Errorf("error while retrieving oldest offset of partition %d: %w", partitionID, err)
	}
	return offset, nil
}

func (c durationKafkaClient) getNewestOffset(topic string, partitionID int32) (int64, error) {
	offset, err := c.kafkaClient.GetOffset(topic, partitionID, sarama.OffsetNewest)
	if err != nil {
		return 0, fmt.Errorf("error while retrieving newest offset of partition %d: %w", partitionID, err)
	}
	return offset, nil
}

func (c durationKafkaClient) getMessageAtOffset(ctx context.Context, topic string, partitionID int32, offset int64) (*sarama.ConsumerMessage, error) {
	pc, err := c.kafkaConsumer.ConsumePartition(topic, partitionID, offset)
	if err != nil {
		if errors.Is(err, sarama.ErrOffsetOutOfRange) {
			closePartitionConsumer(pc)
			return nil, &outOfRangeOffsetError{
				message: err.Error(),
			}
		}

		return nil, fmt.Errorf("error while creating partition consumer on partition %d, offset %d: %w", partitionID, offset, err)
	}

	defer closePartitionConsumer(pc)
	return c.consumeSingleMessage(ctx, pc)
}

func (c durationKafkaClient) consumeSingleMessage(ctx context.Context, pc sarama.PartitionConsumer) (*sarama.ConsumerMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.consumerTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("error while consuming message: %w", ctx.Err())
	case msg := <-pc.Messages():
		return msg, nil
	}
}
