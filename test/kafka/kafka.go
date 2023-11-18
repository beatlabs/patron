package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/component/async"
)

// CreateTopics helper function.
func CreateTopics(broker string, topics ...string) error {
	brk := sarama.NewBroker(broker)

	err := brk.Open(sarama.NewConfig())
	if err != nil {
		return err
	}

	// check if the connection was OK
	connected, err := brk.Connected()
	if err != nil {
		return err
	}
	if !connected {
		return errors.New("not connected")
	}
	deleteReq := &sarama.DeleteTopicsRequest{
		Topics:  topics,
		Timeout: time.Second * 15,
	}

	deleteResp, err := brk.DeleteTopics(deleteReq)
	if err != nil {
		return err
	}

	for k, v := range deleteResp.TopicErrorCodes {
		if v == sarama.ErrNoError || v == sarama.ErrUnknownTopicOrPartition {
			continue
		}
		fmt.Println(k)
		fmt.Println(v)
	}

	time.Sleep(100 * time.Millisecond)

	topicDetail := &sarama.TopicDetail{}
	topicDetail.NumPartitions = int32(1)
	topicDetail.ReplicationFactor = int16(1)
	topicDetail.ConfigEntries = make(map[string]*string)

	topicDetails := make(map[string]*sarama.TopicDetail, len(topics))

	for _, topic := range topics {
		topicDetails[topic] = topicDetail
	}

	request := sarama.CreateTopicsRequest{
		Timeout:      time.Second * 15,
		TopicDetails: topicDetails,
	}

	response, err := brk.CreateTopics(&request)
	if err != nil {
		return err
	}

	for _, val := range response.TopicErrors {
		if val.Err == sarama.ErrTopicAlreadyExists || val.Err == sarama.ErrNoError {
			continue
		}
		return errors.New(val.Error())
	}

	return brk.Close()
}

// NewProducer helper function.
func NewProducer(broker string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	return sarama.NewSyncProducer([]string{broker}, config)
}

// SendMessages to the broker.
func SendMessages(broker string, messages ...*sarama.ProducerMessage) error {
	prod, err := NewProducer(broker)
	if err != nil {
		return err
	}
	err = prod.SendMessages(messages)
	if err != nil {
		return err
	}

	return nil
}

// AsyncConsumeMessages from an async consumer.
func AsyncConsumeMessages(consumer async.Consumer, expectedMessageCount int) ([]string, error) {
	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	ch, chErr, err := consumer.Consume(ctx)
	if err != nil {
		return nil, err
	}

	received := make([]string, 0, expectedMessageCount)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
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

// CreateProducerMessage for a topic.
func CreateProducerMessage(topic, message string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}
}
