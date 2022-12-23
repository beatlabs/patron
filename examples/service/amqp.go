package main

import (
	"context"
	"time"

	"github.com/beatlabs/patron"
	patronamqp "github.com/beatlabs/patron/component/amqp"
	"github.com/beatlabs/patron/log"
	"github.com/streadway/amqp"
)

const (
	amqpURL          = "amqp://guest:guest@localhost:5672/"
	amqpQueue        = "patron"
	amqpExchangeName = "patron"
	amqpExchangeType = amqp.ExchangeFanout
)

func createAMQPConsumer() (patron.Component, error) {
	err := setupQueueAndExchange()
	if err != nil {
		return nil, err
	}

	process := func(ctx context.Context, batch patronamqp.Batch) {
		for _, msg := range batch.Messages() {
			log.FromContext(msg.Context()).Infof("amqp message received:", msg.ID())
			msg.ACK()
		}
	}

	return patronamqp.New(amqpURL, amqpQueue, process, patronamqp.WithRetry(10, 1*time.Second))
}

func setupQueueAndExchange() error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	err = channel.ExchangeDeclare(amqpExchangeName, amqpExchangeType, true, false, false, false, nil)
	if err != nil {
		return err
	}

	q, err := channel.QueueDeclare(amqpQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = channel.QueueBind(q.Name, "", amqpExchangeName, false, nil)
	if err != nil {
		return err
	}
	return nil
}
