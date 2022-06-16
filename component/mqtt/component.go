// Package mqtt provides an instrumented subscriber for MQTT v5.
package mqtt

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/google/uuid"
)

const (
	defaultRetries   = 10
	defaultRetryWait = time.Second
)

type retry struct {
	count uint
	wait  time.Duration
}

type config struct {
	maxMessages *int64
}

// DefaultConfig provides a config with sane default and logging enabled on the callbacks.
func DefaultConfig(brokerURLs []*url.URL, clientID string, router paho.Router) (autopaho.ClientConfig, error) {
	if len(brokerURLs) == 0 {
		return autopaho.ClientConfig{}, errors.New("no broker URLs provided")
	}

	if clientID == "" {
		return autopaho.ClientConfig{}, errors.New("no client id provided")
	}

	if router == nil {
		return autopaho.ClientConfig{}, errors.New("no router provided")
	}

	return autopaho.ClientConfig{
		BrokerUrls:        brokerURLs,
		KeepAlive:         30,
		ConnectRetryDelay: 5 * time.Second,
		ConnectTimeout:    1 * time.Second,
		OnConnectionUp: func(_ *autopaho.ConnectionManager, conAck *paho.Connack) {
			log.Infof("connection is up with reason code: %d\n", conAck.ReasonCode)
		},
		OnConnectError: func(err error) {
			log.Errorf("failed to connect: %v\n", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			Router:   router,
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				log.Warnf("server disconnect received with reason code: %d\n", disconnect.ReasonCode)
			},
			OnClientError: func(err error) {
				log.Errorf("client error occurred: %v\n", err)
			},
			PublishHook: func(publish *paho.Publish) {
				log.Debugf("message published to topic: %s\n", publish.Topic)
			},
		},
	}, nil
}

// Component implementation of an async component.
type Component struct {
	name  string
	cm    *autopaho.ConnectionManager
	cfg   autopaho.ClientConfig
	sub   *paho.Subscribe
	retry retry
}

type Subscription struct {
	name    string
	handler paho.MessageHandler
	options paho.SubscribeOptions
}

// New creates a new component with support for functional configuration.
func New(name string, cfg autopaho.ClientConfig, subs []Subscription, oo ...OptionFunc) (*Component, error) {
	if name == "" {
		return nil, errors.New("component name is empty")
	}

	if len(subs) == 0 {
		return nil, errors.New("subscriptions is empty")
	}

	rt := &paho.StandardRouter{}
	subscribe := &paho.Subscribe{
		Subscriptions: make(map[string]paho.SubscribeOptions, len(subs)),
	}

	for _, sub := range subs {
		// TODO: middleware for observability
		rt.RegisterHandler(sub.name, sub.handler)
		subscribe.Subscriptions[sub.name] = sub.options
	}

	cfg.Router = rt

	cmp := &Component{
		name: name,
		cfg:  cfg,
		sub:  subscribe,
		retry: retry{
			count: defaultRetries,
			wait:  defaultRetryWait,
		},
	}

	for _, optionFunc := range oo {
		err := optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	chErr := make(chan error)

	go c.consume(ctx, chErr)

	for {
		select {
		case err := <-chErr:
			return err
		case <-ctx.Done():
			log.FromContext(ctx).Info("context cancellation received. exiting...")
			return nil
		}
	}
}

func (c *Component) consume(ctx context.Context, chErr chan error) {
	logger := log.FromContext(ctx)

	retries := c.retry.count

	for {
		if ctx.Err() != nil {
			return
		}

		logger.Debug("consume: connecting to broker")

		cm, err := autopaho.NewConnection(ctx, c.cfg)
		if err != nil {
			logger.Errorf("failed to create new connection: %v, sleeping for %v", err, c.retry.wait)
			time.Sleep(c.retry.wait)
			retries--
			if retries > 0 {
				continue
			}
			chErr <- err
			return
		}

		sa, err := cm.Subscribe(ctx, c.sub)
		if err != nil {
			logger.Errorf("failed to subscribe: %v, sleeping for %v", err, c.retry.wait)
			time.Sleep(c.retry.wait)
			retries--
			if retries > 0 {
				continue
			}
			chErr <- err
			return
		}
		logger.Debugf("subscription acknowledgment: %v", sa)

		retries = c.retry.count

		if ctx.Err() != nil {
			return
		}
	}
}

func messageCountInc(queue string, state messageState, hasError bool, count int) {
	hasErrorString := "false"
	if hasError {
		hasErrorString = "true"
	}

	messageCounterVec.WithLabelValues(queue, string(state), hasErrorString).Add(float64(count))
}

func getCorrelationID(ma map[string]*sqs.MessageAttributeValue) string {
	for key, value := range ma {
		if key == correlation.HeaderID {
			if value.StringValue != nil {
				return *value.StringValue
			}
			break
		}
	}
	return uuid.New().String()
}

func mapHeader(ma map[string]*sqs.MessageAttributeValue) map[string]string {
	mp := make(map[string]string)
	for key, value := range ma {
		if value.StringValue != nil {
			mp[key] = *value.StringValue
		}
	}
	return mp
}
