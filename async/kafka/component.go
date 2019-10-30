package kafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/log"
	"strings"
)

// consumerFactoryBuilder contains all the necessary information
// to be used for processing messages from kafka
type consumerFactoryBuilder struct {
	ctx     context.Context
	name    string
	group   string
	ct      string
	topic   string
	brokers string
	cfg     sarama.Config
	process func(message async.Message) error
	dec     func(contentType string) (encoding.DecodeRawFunc, error)
}

// NewConsumerConfig will create a new basic configuration struct
func NewComponentBuilder(ctx context.Context, name string, topic string, brokers string) *consumerFactoryBuilder {
	return &consumerFactoryBuilder{
		ctx:     ctx,
		name:    name,
		topic:   topic,
		brokers: brokers,
	}
}

// SetGroup will set the current group for our kafka consumer
func (cc *consumerFactoryBuilder) SetGroup(group string) *consumerFactoryBuilder {
	cc.group = group
	return cc
}

// SetContentType will set the content type for the kafka consumer
func (cc *consumerFactoryBuilder) SetContentType(ct string) *consumerFactoryBuilder {
	cc.ct = ct
	return cc
}

// ProcessWith will set the processor for the incoming kafka messages
func (cc *consumerFactoryBuilder) ProcessWith(process func(message async.Message) error) *consumerFactoryBuilder {
	cc.process = process
	return cc
}

// SetValueDecoder will provide a decoder for the value part of the kafka message
// it should be expected to decode according to the processor
func (cc *consumerFactoryBuilder) SetValueDecoder(dec func(contentType string) (encoding.DecodeRawFunc, error)) *consumerFactoryBuilder {
	cc.dec = dec
	return cc
}

// checkValues will check and make sure we have all necessary values
// or appropriate void implementation for mandatory parameters,
// otherwise it will return an error
func (cc *consumerFactoryBuilder) checkValues() error {

	if cc.name == "" {
		return errors.New("'name' is mandatory for creating a kafka consumer")
	}

	if cc.process == nil {
		cc.process = func(message async.Message) error {
			log.FromContext(cc.ctx).Infof("Message received on empty processor : %v", message)
			return nil
		}
	}

	log.FromContext(cc.ctx).Infof("Consumer config validated %v", cc)

	return nil

}

func (cc *consumerFactoryBuilder) CreateFactory() (*Factory, error) {
	if err := cc.checkValues(); err != nil {
		return nil, fmt.Errorf("Could not build consumer from %v ,\n %w", cc, err)
	}

	// Create Factory
	return New(cc.name, cc.ct, cc.topic, cc.group, strings.Split(cc.brokers, ","),
		// inject the decoder
		func(c *consumer) error {
			// apply the decoder only if it s not nil,
			// otherwise it will just overwrite our default implementation
			if cc.dec != nil {
				c.dec = cc.dec
			}
			return nil
		})
}
