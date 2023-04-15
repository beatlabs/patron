// Package kafka provides consumer abstractions and base functionality with included tracing capabilities.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
)

const (
	consumerComponent = "kafka-consumer"
	// MessageReceived is used to label the Prometheus Message Status counter.
	MessageReceived = "received"
	// MessageClaimErrors is used to label the Prometheus Message Status counter.
	MessageClaimErrors = "claim-errors"
	// MessageDecoded is used to label the Prometheus Message Status counter.
	MessageDecoded = "decoded"
)

var (
	topicPartitionOffsetDiff *prometheus.GaugeVec
	messageStatus            *prometheus.CounterVec
	messageConfirmation      *prometheus.CounterVec
)

// TopicPartitionOffsetDiffGaugeSet creates a new Gauge that measures partition offsets.
func TopicPartitionOffsetDiffGaugeSet(group, topic string, partition int32, high, offset int64) {
	topicPartitionOffsetDiff.WithLabelValues(group, topic, strconv.FormatInt(int64(partition), 10)).Set(float64(high - offset))
}

// MessageStatusCountInc increments the messageStatus counter for a certain status.
func MessageStatusCountInc(status, group, topic string) {
	messageStatus.WithLabelValues(status, group, topic).Inc()
}

func messageConfirmationCountInc(status string) {
	messageConfirmation.WithLabelValues(status).Inc()
}

func init() {
	topicPartitionOffsetDiff = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "kafka_consumer",
			Name:      "offset_diff",
			Help:      "Message offset difference with high watermark, classified by topic and partition",
		},
		[]string{"group", "topic", "partition"},
	)

	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_consumer",
			Name:      "message_status",
			Help:      "Message status counter (received, decoded, decoding-errors) classified by topic and partition",
		}, []string{"status", "group", "topic"},
	)

	messageConfirmation = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_consumer",
			Name:      "message_confirmation",
			Help:      "Message confirmation counter (ACK/NACK)",
		}, []string{"status"},
	)

	prometheus.MustRegister(
		topicPartitionOffsetDiff,
		messageStatus,
		messageConfirmation,
	)
}

// ConsumerConfig is the common configuration of patron kafka consumers.
type ConsumerConfig struct {
	Brokers                 []string
	Buffer                  int
	DecoderFunc             encoding.DecodeRawFunc
	DurationBasedConsumer   bool
	DurationOffset          time.Duration
	TimeExtractor           func(*sarama.ConsumerMessage) (time.Time, error)
	TimestampBasedConsumer  bool
	TimestampOffset         int64
	SaramaConfig            *sarama.Config
	LatestOffsetReachedChan chan<- struct{}
}

type message struct {
	span opentracing.Span
	ctx  context.Context
	sess sarama.ConsumerGroupSession
	msg  *sarama.ConsumerMessage
	dec  encoding.DecodeRawFunc
}

// Context returns the context encapsulated in the message.
func (m *message) Context() context.Context {
	return m.ctx
}

// Decode will implement the decoding logic in order to transform the message bytes to a business entity.
func (m *message) Decode(v interface{}) error {
	return m.dec(m.msg.Value, v)
}

// Ack sends acknowledgment that the message has been processed.
func (m *message) Ack() error {
	if m.sess != nil {
		m.sess.MarkMessage(m.msg, "")
	}
	messageConfirmationCountInc("ACK")
	trace.SpanSuccess(m.span)
	return nil
}

// Nack signals the producing side an erroring condition or inconsistency.
func (m *message) Nack() error {
	messageConfirmationCountInc("NACK")
	trace.SpanError(m.span)
	return nil
}

// Source returns the kafka topic where the message arrived.
func (m *message) Source() string {
	return m.msg.Topic
}

// Payload returns the message payload.
func (m *message) Payload() []byte {
	return m.msg.Value
}

// Raw returns tha Kafka message.
func (m *message) Raw() interface{} {
	return m.msg
}

// ClaimMessage transforms a sarama.ConsumerMessage to an async.Message.
func ClaimMessage(ctx context.Context, msg *sarama.ConsumerMessage, d encoding.DecodeRawFunc, sess sarama.ConsumerGroupSession) (async.Message, error) {
	slog.Debug("data received", slog.String("topic", msg.Topic))

	corID := getCorrelationID(msg.Headers)

	sp, ctxCh := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, msg.Topic),
		consumerComponent, corID, mapHeader(msg.Headers))
	ctxCh = correlation.ContextWithID(ctxCh, corID)
	ctxCh = log.WithContext(ctxCh, slog.With(slog.String(correlation.ID, corID)))

	dec, err := determineDecoder(d, msg, sp)
	if err != nil {
		return nil, fmt.Errorf("could not determine decoder %w", err)
	}

	return &message{
		ctx:  ctxCh,
		dec:  dec,
		span: sp,
		msg:  msg,
		sess: sess,
	}, nil
}

func determineDecoder(d encoding.DecodeRawFunc, msg *sarama.ConsumerMessage, sp opentracing.Span) (encoding.DecodeRawFunc, error) {
	if d != nil {
		return d, nil
	}

	ct, err := determineContentType(msg.Headers)
	if err != nil {
		trace.SpanError(sp)
		return nil, fmt.Errorf("failed to determine content type from message headers %v : %w", msg.Headers, err)
	}

	dec, err := async.DetermineDecoder(ct)
	if err != nil {
		trace.SpanError(sp)
		return nil, fmt.Errorf("failed to determine decoder from message content type %v %w", ct, err)
	}

	return dec, nil
}

func getCorrelationID(hh []*sarama.RecordHeader) string {
	for _, h := range hh {
		if string(h.Key) == correlation.HeaderID {
			if len(h.Value) > 0 {
				return string(h.Value)
			}
			break
		}
	}
	return uuid.New().String()
}

func determineContentType(hdr []*sarama.RecordHeader) (string, error) {
	for _, h := range hdr {
		if string(h.Key) == encoding.ContentTypeHeader {
			return string(h.Value), nil
		}
	}
	return "", errors.New("content type header is missing")
}

func mapHeader(hh []*sarama.RecordHeader) map[string]string {
	mp := make(map[string]string)
	for _, h := range hh {
		mp[string(h.Key)] = string(h.Value)
	}
	return mp
}
