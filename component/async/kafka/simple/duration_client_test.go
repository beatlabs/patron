package simple

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Consumer_GetTimeBasedOffsetsPerPartition(t *testing.T) {
	topic := "topic"
	since := mustCreateTime(t, "2020-01-01T12:00:00Z")
	dummyClient := client(topic).partitionIDs([]int32{0}, nil).build()
	// The message is invalid as the time extractor we use require a timestamp header
	invalidMessage := &sarama.ConsumerMessage{}

	testCases := map[string]struct {
		globalTimeout   time.Duration
		client          *clientMock
		expectedOffsets map[int32]int64
		expectedErr     error
	}{
		"success - multiple partitions": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0, 1, 2}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since},
						},
					},
				}).
				partition(1, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						0: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-3 * time.Hour)},
						},
						1: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-2 * time.Hour)},
						},
						2: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-1 * time.Hour)},
						},
						3: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(1 * time.Hour)},
						},
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(2 * time.Hour)},
						},
					},
				}).
				partition(2, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-4 * time.Hour)},
						},
						5: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-3 * time.Hour)},
						},
						6: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-2 * time.Hour)},
						},
						7: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-1 * time.Hour)},
						},
						8: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(1 * time.Hour)},
						},
						9: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(2 * time.Hour)},
						},
					},
				}).build(),
			expectedOffsets: map[int32]int64{
				0: 4,
				1: 3,
				2: 8,
			},
		},
		"success - all inside": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						0: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(1 * time.Hour)},
						},
						1: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(2 * time.Hour)},
						},
						2: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(3 * time.Hour)},
						},
						3: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(4 * time.Hour)},
						},
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(5 * time.Hour)},
						},
					},
				}).build(),
			expectedOffsets: map[int32]int64{
				0: 0,
			},
		},
		"success - all outside": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-6 * time.Hour)},
						},
						5: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-5 * time.Hour)},
						},
						6: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-4 * time.Hour)},
						},
						7: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-3 * time.Hour)},
						},
						8: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-2 * time.Hour)},
						},
						9: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-1 * time.Hour)},
						},
					},
				}).build(),
			expectedOffsets: map[int32]int64{
				0: 10,
			},
		},
		"success - out of range offset": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since},
							err: &outOfRangeOffsetError{
								message: "foo",
							},
						},
						5: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-3 * time.Hour)},
						},
						6: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-2 * time.Hour)},
						},
						7: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(-1 * time.Hour)},
						},
						8: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(1 * time.Hour)},
						},
						9: {
							msg: &sarama.ConsumerMessage{Timestamp: since.Add(2 * time.Hour)},
						},
					},
				}).build(),
			expectedOffsets: map[int32]int64{
				0: 8,
			},
		},
		"error - get partitions": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs(nil, errors.New("foo")).build(),
			expectedErr: errors.New("foo"),
		},
		"error - timeout": {
			globalTimeout: time.Nanosecond,
			client:        dummyClient,
			expectedErr:   errors.New("context cancelled before collecting partition responses: context deadline exceeded"),
		},
		"error - get oldest offset": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{err: errors.New("foo")},
				}).build(),
			expectedErr: errors.New("foo"),
		},
		"error - get newest offset": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{err: errors.New("foo")},
				}).build(),
			expectedErr: errors.New("foo"),
		},
		"error - invalid message": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: invalidMessage,
						},
					},
				}).build(),
			expectedErr: errors.New("error while executing comparator: empty time"),
		},
		"error - get message offset": {
			globalTimeout: time.Second,
			client: client(topic).
				partitionIDs([]int32{0}, nil).
				partition(0, partitionConfig{
					oldest: offset{offset: 0},
					newest: offset{offset: 10},
					messages: map[int64]messageAtOffset{
						4: {
							msg: &sarama.ConsumerMessage{Timestamp: since},
							err: errors.New("foo"),
						},
					},
				}).build(),
			expectedErr: errors.New("error while retrieving message offset 4 on partition 0: foo"),
		},
	}
	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			consumer, err := newDurationClient(tt.client)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), tt.globalTimeout)
			defer cancel()

			offsets, err := consumer.getTimeBasedOffsetsPerPartition(ctx, topic, since, kafkaHeaderTimeExtractor)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOffsets, offsets)
			}
		})
	}
}

type clientBuilder struct {
	topic            string
	partitions       []int32
	partitionsError  error
	partitionConfigs map[int32]partitionConfig
}

func client(topic string) *clientBuilder {
	return &clientBuilder{
		topic:            topic,
		partitionConfigs: make(map[int32]partitionConfig),
	}
}

func (c *clientBuilder) partitionIDs(partitions []int32, err error) *clientBuilder {
	c.partitions = partitions
	c.partitionsError = err
	return c
}

type partitionConfig struct {
	oldest   offset
	newest   offset
	messages map[int64]messageAtOffset
}

type offset struct {
	offset int64
	err    error
}

type messageAtOffset struct {
	msg *sarama.ConsumerMessage
	err error
}

func (c *clientBuilder) partition(partitionID int32, partitionConfig partitionConfig) *clientBuilder {
	c.partitionConfigs[partitionID] = partitionConfig
	return c
}

func (c *clientBuilder) build() *clientMock {
	return &clientMock{
		builder: c,
	}
}

type clientMock struct {
	builder *clientBuilder
}

func (c *clientMock) getPartitionIDs(_ string) ([]int32, error) {
	return c.builder.partitions, c.builder.partitionsError
}

func (c *clientMock) getOldestOffset(_ string, partitionID int32) (int64, error) {
	return c.builder.partitionConfigs[partitionID].oldest.offset, c.builder.partitionConfigs[partitionID].oldest.err
}

func (c *clientMock) getNewestOffset(_ string, partitionID int32) (int64, error) {
	return c.builder.partitionConfigs[partitionID].newest.offset, c.builder.partitionConfigs[partitionID].newest.err
}

func (c *clientMock) getMessageAtOffset(_ context.Context, _ string, partitionID int32, offset int64) (*sarama.ConsumerMessage, error) {
	cfg := c.builder.partitionConfigs[partitionID].messages[offset]
	return cfg.msg, cfg.err
}

func mustCreateTime(t *testing.T, timestamp string) time.Time {
	ts, err := time.Parse(time.RFC3339, timestamp)
	require.NoError(t, err)
	return ts
}

func kafkaHeaderTimeExtractor(msg *sarama.ConsumerMessage) (time.Time, error) {
	if msg.Timestamp.IsZero() {
		return time.Time{}, errors.New("empty time")
	}
	return msg.Timestamp, nil
}
