// Package sns provides a set of common interfaces and structs for publishing messages to AWS SNS. Implementations
// in this package also include distributed tracing capabilities by default.
//
// Deprecated: The SNS client package is superseded by the `github.com/beatlabs/client/sns/v2` package.
// Please refer to the documents and the examples for the usage.
//
// This package is frozen and no new functionality will be added.
package sns

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	publisherComponent = "sns-publisher"
)

// Publisher is the interface defining an SNS publisher, used to publish messages to SNS.
type Publisher interface {
	Publish(ctx context.Context, msg Message) (messageID string, err error)
}

// TracedPublisher is an implementation of the Publisher interface with added distributed tracing capabilities.
type TracedPublisher struct {
	api snsiface.SNSAPI

	// component is the name of the component used in tracing operations
	component string
	// tag is the base tag used during tracing operations
	tag opentracing.Tag
}

// NewPublisher creates a new SNS publisher.
//
// Deprecated: The SNS client package is superseded by the `github.com/beatlabs/client/sns/v2` package.
// Please refer to the documents and the examples for the usage.
//
// This package is frozen and no new functionality will be added.
func NewPublisher(api snsiface.SNSAPI) (*TracedPublisher, error) {
	if api == nil {
		return nil, errors.New("missing api")
	}

	return &TracedPublisher{
		api:       api,
		component: publisherComponent,
		tag:       ext.SpanKindProducer,
	}, nil
}

// Publish tries to publish a new message to SNS. It also stores tracing information.
func (p TracedPublisher) Publish(ctx context.Context, msg Message) (messageID string, err error) {
	span, _ := trace.ChildSpan(ctx, p.publishOpName(msg), p.component, ext.SpanKindProducer, p.tag)

	carrier := snsHeadersCarrier{}
	err = span.Tracer().Inject(span.Context(), opentracing.TextMap, &carrier)
	if err != nil {
		return "", fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	msg.injectHeaders(carrier)
	msg.setMessageAttribute(correlation.HeaderID, correlation.IDFromContext(ctx))

	out, err := p.api.PublishWithContext(ctx, msg.input)

	trace.SpanComplete(span, err)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
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
		msg.tracingTarget(),
	)
}

type snsHeadersCarrier map[string]interface{}

// Set implements Set() of opentracing.TextMapWriter.
func (c snsHeadersCarrier) Set(key, val string) {
	c[key] = val
}
