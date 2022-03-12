package test

import (
	"context"
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
)

// CreateTopics helper function.
func CreateTopics(broker string, topics ...string) error {
	brk := sarama.NewBroker(broker)
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0

	err := brk.Open(config)
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

	_, err = brk.DeleteTopics(deleteReq)
	if err != nil {
		return err
	}

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

	t := response.TopicErrors
	for _, val := range t {
		if val.Err == sarama.ErrTopicAlreadyExists || val.Err == sarama.ErrNoError {
			continue
		}
		msg := val.Error()
		if msg != "" {
			return errors.New(msg)
		}
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
	for _, message := range messages {
		_, _, err = prod.SendMessage(message)
		if err != nil {
			return err
		}
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
