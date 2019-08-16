package sns

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Publisher is the interface defining an SNS publisher.
type Publisher interface {
	Publish(msg Message) (messageID string, err error)
}

// TracedPublisher is the SNS publisher component.
type TracedPublisher struct {
	api snsiface.SNSAPI

	// component is the name of the component used in tracing operations
	component string
	// tag is the base tag used during tracing operations
	tag opentracing.Tag
}

// NewPublisher creates a new SNS publisher.
func NewPublisher(api snsiface.SNSAPI) (*TracedPublisher, error) {
	if api == nil {
		return nil, errors.New("missing api")
	}

	return &TracedPublisher{
		api:       api,
		component: trace.SNSPublisherComponent,
		tag:       ext.SpanKindProducer,
	}, nil
}

// Publish tries to publish a new message to SNS, with an added tracing capability.
func (p TracedPublisher) Publish(ctx context.Context, msg Message) (messageID string, err error) {
	span, _ := trace.ChildSpan(ctx, p.publishOpName(msg), p.component, p.tag)
	out, err := p.api.Publish(msg.input)

	if err != nil {
		trace.SpanError(span)
		return "", errors.Wrap(err, "failed to publish message")
	}

	if out.MessageId == nil {
		return "", errors.New("tried to publish a message but no message ID returned")
	}

	return *out.MessageId, nil
}

// publishOpName returns the publish operation name based on the message.
func (p TracedPublisher) publishOpName(msg Message) string {
	return trace.ComponentOpName(
		p.component,
		fmt.Sprintf("publish:%s", msg.tracingTarget()),
	)
}
