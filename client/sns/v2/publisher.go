// Package v2 provides a wrapper for publishing messages to AWS SNS. Implementations
// in this package also include distributed tracing capabilities by default.
package v2

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	attributeDataTypeString = "String"

	publisherComponent = "sns-publisher"

	tracingTargetUnknown   = "unknown"
	tracingTargetTopicArn  = "topic-arn"
	tracingTargetTargetArn = "target-arn"
)

var publishDurationMetrics *prometheus.HistogramVec

func init() {
	publishDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "sns",
			Name:      "publish_duration_seconds",
			Help:      "AWS SNS publish completed by the client.",
		},
		[]string{"topic", "success"},
	)
	prometheus.MustRegister(publishDurationMetrics)
}

// Publisher is an implementation of the Publisher interface with added distributed tracing capabilities.
type Publisher struct {
	api snsiface.SNSAPI
}

// New creates a new SNS publisher.
func New(api snsiface.SNSAPI) (Publisher, error) {
	if api == nil {
		return Publisher{}, errors.New("missing api")
	}

	return Publisher{api: api}, nil
}

// Publish tries to publish a new message to SNS. It also stores tracing information.
func (p Publisher) Publish(ctx context.Context, input *sns.PublishInput) (messageID string, err error) {
	span, _ := trace.ChildSpan(ctx, trace.ComponentOpName(publisherComponent, tracingTarget(input)), publisherComponent, ext.SpanKindProducer)

	if err := injectHeaders(span, input); err != nil {
		log.FromContext(ctx).Warnf("failed to inject tracing header: %v", err)
	}

	start := time.Now()
	out, err := p.api.PublishWithContext(ctx, input)
	if input.TopicArn != nil {
		observePublish(ctx, span, start, *input.TopicArn, err)
	}
	if input.TargetArn != nil {
		observePublish(ctx, span, start, *input.TargetArn, err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	if out.MessageId == nil {
		return "", errors.New("tried to publish a message but no message ID returned")
	}

	return *out.MessageId, nil
}

type snsHeadersCarrier map[string]interface{}

// Set implements Set() of opentracing.TextMapWriter.
func (c snsHeadersCarrier) Set(key, val string) {
	c[key] = val
}

func tracingTarget(input *sns.PublishInput) string {
	if input.TopicArn != nil {
		return fmt.Sprintf("%s:%s", tracingTargetTopicArn, aws.StringValue(input.TopicArn))
	}

	if input.TargetArn != nil {
		return fmt.Sprintf("%s:%s", tracingTargetTargetArn, aws.StringValue(input.TargetArn))
	}

	return tracingTargetUnknown
}

// injectHeaders injects the SNS headers carrier's headers into the message's attributes.
func injectHeaders(span opentracing.Span, input *sns.PublishInput) error {
	if input.MessageAttributes == nil {
		input.MessageAttributes = make(map[string]*sns.MessageAttributeValue)
	}

	carrier := snsHeadersCarrier{}
	if err := span.Tracer().Inject(span.Context(), opentracing.TextMap, &carrier); err != nil {
		return fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	for k, v := range carrier {
		input.MessageAttributes[k] = &sns.MessageAttributeValue{
			DataType:    aws.String(attributeDataTypeString),
			StringValue: aws.String(v.(string)),
		}
	}
	return nil
}

func observePublish(ctx context.Context, span opentracing.Span, start time.Time, topic string, err error) {
	trace.SpanComplete(span, err)

	durationHistogram := trace.Histogram{
		Observer: publishDurationMetrics.WithLabelValues(topic, strconv.FormatBool(err == nil)),
	}
	durationHistogram.Observe(ctx, time.Since(start).Seconds())
}
