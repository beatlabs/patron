package simple

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/beatlabs/patron/log"
)

type durationClient struct {
	client durationKafkaClientAPI
}

func newDurationClient(client durationKafkaClientAPI) (durationClient, error) {
	if client == nil {
		return durationClient{}, errors.New("empty client api")
	}
	return durationClient{client: client}, nil
}

func (d durationClient) getTimeBasedOffsetsPerPartition(ctx context.Context, topic string, since time.Time, timeExtractor TimeExtractor) (map[int32]int64, error) {
	partitionIDs, err := d.client.getPartitionIDs(topic)
	if err != nil {
		return nil, err
	}

	responseCh := make(chan partitionOffsetResponse, len(partitionIDs))
	d.triggerWorkers(ctx, topic, since, timeExtractor, partitionIDs, responseCh)
	return d.aggregateResponses(ctx, partitionIDs, responseCh)
}

type partitionOffsetResponse struct {
	partitionID int32
	offset      int64
	err         error
}

func (d durationClient) triggerWorkers(ctx context.Context, topic string, since time.Time, timeExtractor TimeExtractor, partitionIDs []int32, responseCh chan<- partitionOffsetResponse) {
	for _, partitionID := range partitionIDs {
		partitionID := partitionID
		go func() {
			offset, err := d.getTimeBasedOffset(ctx, topic, since, partitionID, timeExtractor)
			select {
			case <-ctx.Done():
				return
			case responseCh <- partitionOffsetResponse{
				partitionID: partitionID,
				offset:      offset,
				err:         err,
			}:
			}
		}()
	}
}

func (d durationClient) aggregateResponses(ctx context.Context, partitionIDs []int32, responseCh <-chan partitionOffsetResponse) (map[int32]int64, error) {
	numberOfPartitions := len(partitionIDs)
	offsets := make(map[int32]int64, numberOfPartitions)
	numberOfResponses := 0
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled before collecting partition responses: %w", ctx.Err())
		case response := <-responseCh:
			if response.err != nil {
				return nil, response.err
			}

			offsets[response.partitionID] = response.offset
			numberOfResponses++
			if numberOfResponses == numberOfPartitions {
				return offsets, nil
			}
		}
	}
}

func (d durationClient) getTimeBasedOffset(ctx context.Context, topic string, since time.Time, partitionID int32, timeExtractor TimeExtractor) (int64, error) {
	left, err := d.client.getOldestOffset(topic, partitionID)
	if err != nil {
		return 0, err
	}

	newestOffset, err := d.client.getNewestOffset(topic, partitionID)
	if err != nil {
		return 0, err
	}
	// The right boundary must be inclusive
	right := newestOffset - 1

	return d.offsetBinarySearch(ctx, topic, since, partitionID, timeExtractor, left, right)
}

func (d durationClient) offsetBinarySearch(ctx context.Context, topic string, since time.Time, partitionID int32, timeExtractor TimeExtractor, left, right int64) (int64, error) {
	for left <= right {
		mid := left + (right-left)/2

		msg, err := d.client.getMessageAtOffset(ctx, topic, partitionID, mid)
		if err != nil {
			// Under extraordinary circumstances (e.g. the retention policy being applied just before retrieving the message at a particular offset),
			// the offset might not be accessible anymore.
			// In this case, we simply log a warning and restrict the interval to the right.
			if errors.Is(err, &outOfRangeOffsetError{}) {
				log.Warnf("offset %d on partition %d is out of range: %v", mid, partitionID, err)
				left = mid + 1
				continue
			}
			return 0, fmt.Errorf("error while retrieving message offset %d on partition %d: %w", mid, partitionID, err)
		}

		t, err := timeExtractor(msg)
		if err != nil {
			return 0, fmt.Errorf("error while executing comparator: %w", err)
		}

		if t.Equal(since) {
			return mid, nil
		}
		if t.Before(since) {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return left, nil
}
