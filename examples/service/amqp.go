// Package main provides a service for AMQP.
package main

import (
	"context"
	"time"

	"github.com/beatlabs/patron"
	patronamqp "github.com/beatlabs/patron/component/amqp"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func createAMQPConsumer() (patron.Component, error) {
	err := setupQueueAndExchange()
	if err != nil {
		return nil, err
	}

	process := func(_ context.Context, batch patronamqp.Batch) {
		for _, msg := range batch.Messages() {
			err := msg.ACK()
			if err != nil {
				log.FromContext(msg.Context()).Info("amqp message received but ack failed", "msgID", msg.ID(), "error", err)
				continue
			}
			log.FromContext(msg.Context()).Info("amqp message received and acked", "msgID", msg.ID())
		}
	}

	return patronamqp.New(examples.AMQPURL, examples.AMQPQueue, process, patronamqp.WithRetry(10, 1*time.Second))
}

func setupQueueAndExchange() error {
	conn, err := amqp.Dial(examples.AMQPURL)
	if err != nil {
		return err
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	err = channel.ExchangeDeclare(examples.AMQPExchangeName, examples.AMQPExchangeType, true, false, false, false, nil)
	if err != nil {
		return err
	}

	q, err := channel.QueueDeclare(examples.AMQPQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = channel.QueueBind(q.Name, "", examples.AMQPExchangeName, false, nil)
	if err != nil {
		return err
	}
	return nil
}
