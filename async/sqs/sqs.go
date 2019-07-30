package sqs

import (
	"context"
	"github.com/beatlabs/patron/async"
)

type message struct {
}

func (m *message) Context() context.Context {
	panic("implement me")
}

func (m *message) Decode(v interface{}) error {
	panic("implement me")
}

func (m *message) Ack() error {
	panic("implement me")
}

func (m *message) Nack() error {
	panic("implement me")
}

type Factory struct {
}

func (f *Factory) Create() (async.Consumer, error) {
	panic("implement me")
}

type consumer struct {
}

func (c *consumer) Consume(context.Context) (<-chan async.Message, <-chan error, error) {
	panic("implement me")
}

func (c *consumer) Close() error {
	panic("implement me")
}
