package sns

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

// Publisher is the interface defining an SNS publisher.
type Publisher interface {
	Publish(msg Message) (messageID string, err error)
}

// TracedPublisher is the SNS publisher component.
type TracedPublisher struct {
	sns snsiface.SNSAPI
}

// NewPublisher creates a new SNS publisher.
func NewPublisher(cfg Config) (*TracedPublisher, error) {
	sns := sns.New(cfg.sess, cfg.cfgs...)

	return &TracedPublisher{
		sns: sns,
	}, nil
}

// Publish tries to publish a new message to SNS.
func (p TracedPublisher) Publish(msg Message) (messageID string, err error) {
	out, err := p.sns.Publish(
		msg.input,
		// Subject:  aws.String("my test subject"),
		// Message:  aws.String("my test message"),
		// TopicArn: aws.String("my test topic"),
	)
	if err != nil {
		// TODO: log it
		return "", errors.New("could not publish the message")
	}
	if out.MessageId == nil {
		// TODO: log it
		return "", errors.New("no message ID TODO: fix this error message")
	}
	return *out.MessageId, nil
}
