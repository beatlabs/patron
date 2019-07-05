package kafka

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

var topicPartitionOffsetDiff *prometheus.GaugeVec

func topicPartitionOffsetDiffGaugeSet(group, topic string, partition int32, high, offset int64) {
	topicPartitionOffsetDiff.WithLabelValues(group, topic, strconv.FormatInt(int64(partition), 10)).Set(float64(high - offset))
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
	prometheus.MustRegister(topicPartitionOffsetDiff)
}

type message struct {
	span opentracing.Span
	ctx  context.Context
	sess sarama.ConsumerGroupSession
	msg  *sarama.ConsumerMessage
	dec  encoding.DecodeRawFunc
}

func (m *message) Context() context.Context {
	return m.ctx
}

func (m *message) Decode(v interface{}) error {
	return m.dec(m.msg.Value, v)
}

func (m *message) Ack() error {
	if m.sess != nil {
		m.sess.MarkMessage(m.msg, "")
	}
	trace.SpanSuccess(m.span)
	return nil
}

func (m *message) Nack() error {
	trace.SpanError(m.span)
	return nil
}

// Factory definition of a consumer factory.
type Factory struct {
	name    string
	ct      string
	topic   string
	group   string
	brokers []string
	oo      []OptionFunc
}

// New constructor.
func New(name, ct, topic, group string, brokers []string, oo ...OptionFunc) (*Factory, error) {

	if name == "" {
		return nil, errors.New("name is required")
	}

	if len(brokers) == 0 {
		return nil, errors.New("provide at least one broker")
	}

	if topic == "" {
		return nil, errors.New("topic is required")
	}

	return &Factory{name: name, ct: ct, topic: topic, group: group, brokers: brokers, oo: oo}, nil
}

// Create a new consumer.
func (f *Factory) Create() (async.Consumer, error) {

	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	config := sarama.NewConfig()
	config.ClientID = fmt.Sprintf("%s-%s", host, f.name)
	config.Consumer.Return.Errors = true
	config.Version = sarama.V0_11_0_0

	c := &consumer{
		brokers:     f.brokers,
		topic:       f.topic,
		cfg:         config,
		contentType: f.ct,
		buffer:      1000,
		info:        make(map[string]interface{}),
	}

	if f.group != "" {
		c.group = f.group
		c.traceTag = opentracing.Tag{Key: "group", Value: f.group}
		c.buffer = 0
	}

	for _, o := range f.oo {
		err = o(c)
		if err != nil {
			return nil, err
		}
	}

	c.createInfo()
	return c, nil
}

type consumer struct {
	brokers     []string
	topic       string
	group       string
	buffer      int
	traceTag    opentracing.Tag
	cfg         *sarama.Config
	contentType string
	cnl         context.CancelFunc
	cg          sarama.ConsumerGroup
	ms          sarama.Consumer
	info        map[string]interface{}
}

// Info return the information of the consumer.
func (c *consumer) Info() map[string]interface{} {
	return c.info
}

// Consume starts consuming messages from a Kafka topic.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	ctx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	if c.group != "" {
		return consumeWithGroup(ctx, c)
	}

	return consume(ctx, c)
}

func consumeWithGroup(ctx context.Context, c *consumer) (<-chan async.Message, <-chan error, error) {

	cg, err := sarama.NewConsumerGroup(c.brokers, c.group, c.cfg)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create consumer")
	}
	c.cg = cg
	log.Infof("consuming messages from topic '%s' using group '%s'", c.topic, c.group)

	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)

	go func(consumer sarama.ConsumerGroup) {
		for {
			select {
			case <-ctx.Done():
				log.Info("canceling consuming messages requested")
				closeConsumer(consumer)
				return
			case consumerError := <-consumer.Errors():
				closeConsumer(consumer)
				chErr <- consumerError
				return
			}
		}
	}(c.cg)

	// Iterate over consumer sessions.
	go func(consumer sarama.ConsumerGroup) {
		topics := []string{c.topic}
		handler := handler{consumer: c, messages: chMsg}
		for {
			err := consumer.Consume(ctx, topics, handler)
			if err != nil {
				chErr <- err
			}
		}
	}(c.cg)

	return chMsg, chErr, nil
}

func consume(ctx context.Context, c *consumer) (<-chan async.Message, <-chan error, error) {

	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)

	log.Infof("consuming messages from topic '%s' without using consumer group", c.topic)
	pcs, err := c.partitions()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get partitions")
	}
	// When kafka cluster is not fully initialized, we may get 0 partions.
	if len(pcs) == 0 {
		return nil, nil, errors.New("got 0 partitions")
	}

	for _, pc := range pcs {
		go func(consumer sarama.PartitionConsumer) {
			for {
				select {
				case <-ctx.Done():
					log.Info("canceling consuming messages requested")
					closePartitionConsumer(consumer)
					return
				case consumerError := <-consumer.Errors():
					closePartitionConsumer(consumer)
					chErr <- consumerError
					return
				case m := <-consumer.Messages():
					topicPartitionOffsetDiffGaugeSet("", m.Topic, m.Partition, consumer.HighWaterMarkOffset(), m.Offset)

					go func() {
						msg, err := claimMessage(ctx, c, m)
						if err != nil {
							chErr <- err
							return
						}
						chMsg <- msg
					}()
				}
			}
		}(pc)
	}

	return chMsg, chErr, nil

}

func claimMessage(ctx context.Context, c *consumer, msg *sarama.ConsumerMessage) (*message, error) {
	log.Debugf("data received from topic %s", msg.Topic)

	sp, chCtx := trace.ConsumerSpan(
		ctx,
		trace.ComponentOpName(trace.KafkaConsumerComponent, msg.Topic),
		trace.KafkaConsumerComponent,
		mapHeader(msg.Headers),
	)
	var ct string
	var err error
	if c.contentType != "" {
		ct = c.contentType
	} else {
		ct, err = determineContentType(msg.Headers)
		if err != nil {
			trace.SpanError(sp)
			return nil, errors.Wrap(err, "failed to determine content type")
		}
	}

	dec, err := async.DetermineDecoder(ct)
	if err != nil {
		trace.SpanError(sp)
		return nil, errors.Wrapf(err, "failed to determine decoder for %s", ct)
	}

	chCtx = log.WithContext(chCtx, log.Sub(map[string]interface{}{"messageID": uuid.New().String()}))

	return &message{
		ctx:  chCtx,
		dec:  dec,
		span: sp,
		msg:  msg,
	}, nil
}

// Close handles closing consumer.
func (c *consumer) Close() error {
	if c.cnl != nil {
		c.cnl()
	}

	if c.cg == nil {
		return nil
	}

	return errors.Wrap(c.cg.Close(), "failed to close consumer")
}

func (c *consumer) partitions() ([]sarama.PartitionConsumer, error) {

	ms, err := sarama.NewConsumer(c.brokers, c.cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create consumer")
	}
	c.ms = ms

	partitions, err := c.ms.Partitions(c.topic)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get partitions")
	}

	pcs := make([]sarama.PartitionConsumer, len(partitions))

	for i, partition := range partitions {

		pc, err := c.ms.ConsumePartition(c.topic, partition, c.cfg.Consumer.Offsets.Initial)
		if nil != err {
			return nil, errors.Wrap(err, "failed to get partition consumer")
		}
		pcs[i] = pc
	}

	return pcs, nil
}

func (c *consumer) createInfo() {
	c.info["type"] = "kafka-consumer"
	c.info["brokers"] = strings.Join(c.brokers, ",")
	c.info["group"] = c.group
	c.info["topic"] = c.topic
	c.info["buffer"] = c.buffer
	c.info["default-content-type"] = c.contentType
}

type handler struct {
	consumer *consumer
	messages chan async.Message
}

func (h handler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h handler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h handler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := sess.Context()
	for msg := range claim.Messages() {
		topicPartitionOffsetDiffGaugeSet(h.consumer.group, msg.Topic, msg.Partition, claim.HighWaterMarkOffset(), msg.Offset)
		m, err := claimMessage(ctx, h.consumer, msg)
		if err != nil {
			return err
		}
		m.sess = sess
		h.messages <- m
	}
	return nil
}

func closeConsumer(cns sarama.ConsumerGroup) {
	if cns == nil {
		return
	}
	err := cns.Close()
	if err != nil {
		log.Errorf("failed to close partition consumer: %v", err)
	}
}

func closePartitionConsumer(cns sarama.PartitionConsumer) {
	if cns == nil {
		return
	}
	err := cns.Close()
	if err != nil {
		log.Errorf("failed to close partition consumer: %v", err)
	}
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
