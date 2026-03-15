// Package kafka provides a consumer component for processing Kafka messages using franz-go.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/observability/log"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"github.com/twmb/franz-go/plugin/kslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// FailStrategy type definition.
type FailStrategy int

const (
	// ExitStrategy does not commit failed message offsets and exits the application.
	ExitStrategy FailStrategy = iota
	// SkipStrategy commits the offset of messages that failed processing, and continues processing.
	SkipStrategy
)

// ProcessorFunc definition of a batch processor function.
// The ctx parameter carries lifecycle signals (shutdown, deadlines).
// Each record's own Context (rec.Context) carries per-record trace/log/correlation context.
type ProcessorFunc func(ctx context.Context, records []*kgo.Record) error

type recordSpan struct {
	record *kgo.Record
	span   trace.Span
}

const (
	defaultRetries         = 3
	defaultRetryWait       = 10 * time.Second
	defaultBatchSize       = 1
	defaultBatchTimeout    = 100 * time.Millisecond
	defaultFailureStrategy = ExitStrategy
)

// New initializes a new kafka consumer component with support for functional configuration.
// The default failure strategy is the ExitStrategy.
// The default batch size is 1 and the batch timeout is 100ms.
// The default number of retries is 3 and the retry wait is 10s.
// The default commit strategy uses AutoCommitMarks with MarkCommitRecords after successful processing.
func New(name, group string, brokers, topics []string, proc ProcessorFunc, opts []kgo.Opt, oo ...OptionFunc) (*Component, error) {
	var errs []error
	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if group == "" {
		errs = append(errs, errors.New("consumer group is required"))
	}

	if validation.IsStringSliceEmpty(brokers) {
		errs = append(errs, errors.New("brokers are empty or have an empty value"))
	}

	if validation.IsStringSliceEmpty(topics) {
		errs = append(errs, errors.New("topics are empty or have an empty value"))
	}

	if proc == nil {
		errs = append(errs, errors.New("work processor is required"))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	cmp := &Component{
		name:         name,
		group:        group,
		brokers:      brokers,
		topics:       topics,
		proc:         proc,
		retries:      defaultRetries,
		retryWait:    defaultRetryWait,
		batchSize:    defaultBatchSize,
		batchTimeout: defaultBatchTimeout,
		failStrategy: defaultFailureStrategy,
		opts:         opts,
	}

	for _, optionFunc := range oo {
		err := optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Component is a kafka consumer implementation that processes messages in batch.
type Component struct {
	name                      string
	group                     string
	topics                    []string
	brokers                   []string
	opts                      []kgo.Opt
	proc                      ProcessorFunc
	failStrategy              FailStrategy
	batchSize                 uint
	batchTimeout              time.Duration
	batchMessageDeduplication bool
	retries                   uint32
	retryWait                 time.Duration
	manualCommit              bool
	sessionCallback           func() error
}

// Run starts the consumer processing loop to process messages from Kafka.
func (c *Component) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return c.processing(ctx)
}

func (c *Component) processing(ctx context.Context) error {
	var componentError error

	retries := c.retries
	for i := uint32(0); i <= retries; i++ {
		tracer := kotel.NewTracer(kotel.TracerProvider(otel.GetTracerProvider()), kotel.ConsumerGroup(c.group))
		meter := kotel.NewMeter(kotel.MeterProvider(otel.GetMeterProvider()))
		kotelService := kotel.NewKotel(kotel.WithTracer(tracer), kotel.WithMeter(meter))

		handler := newConsumerHandler(ctx, c.name, c.group, tracer, c.proc, c.failStrategy, c.batchSize,
			c.batchTimeout, c.manualCommit, c.batchMessageDeduplication)

		opts := []kgo.Opt{
			kgo.SeedBrokers(c.brokers...),
			kgo.ConsumerGroup(c.group),
			kgo.ConsumeTopics(c.topics...),
			kgo.WithHooks(kotelService.Hooks()...),
			kgo.WithLogger(kslog.New(slog.Default())),
		}

		if c.manualCommit {
			opts = append(opts, kgo.DisableAutoCommit(), kgo.BlockRebalanceOnPoll())
		} else {
			opts = append(opts, kgo.AutoCommitMarks())
		}

		if c.sessionCallback != nil {
			opts = append(opts, kgo.OnPartitionsAssigned(func(context.Context, *kgo.Client, map[string][]int32) {
				err := c.sessionCallback()
				if err != nil {
					slog.Error("error executing session callback", log.ErrorAttr(err))
					handler.setErr(err)
				}
			}))
		}

		opts = append(opts, c.opts...)

		cl, err := kgo.NewClient(opts...)
		componentError = err
		if err != nil {
			slog.Error("error creating kafka consumer client", log.ErrorAttr(err))
		}

		if cl != nil {
			slog.Debug("consuming messages", slog.Any("topics", c.topics), slog.String("group", c.group))

			err = handler.consume(ctx, cl)
			componentError = err
			if err != nil {
				slog.Error("failure from kafka consumer", log.ErrorAttr(err))
			}

			if handler.getErr() != nil && componentError == nil {
				componentError = handler.getErr()
			}

			cl.Close()
		}

		consumerErrorsInc(ctx, c.name)

		if c.retries > 0 {
			if handler.processedMessages {
				i = 0
			}

			if componentError == nil {
				componentError = handler.getErr()
			}

			slog.Error("failed run", slog.Uint64("current", uint64(i)), slog.Uint64("retries", uint64(c.retries)),
				slog.Duration("wait", c.retryWait), log.ErrorAttr(componentError))
			time.Sleep(c.retryWait)

			if i < retries {
				componentError = nil
			}
		}

		if i == retries && componentError == nil && handler.getErr() != nil {
			componentError = fmt.Errorf("message processing failure exhausted %d retries: %w", i, handler.getErr())
		}
	}

	return componentError
}

type consumerHandler struct {
	ctx context.Context

	name  string
	group string

	// kotel tracer for creating process spans
	kotelTracer *kotel.Tracer

	// buffer
	batchSize                 uint
	ticker                    *time.Ticker
	batchMessageDeduplication bool

	// callback
	proc ProcessorFunc

	// failures strategy
	failStrategy FailStrategy

	// committing manually after every batch
	manualCommit bool

	// lock to protect buffer operation
	mu     sync.RWMutex
	recBuf []*kgo.Record

	// processing error
	err error

	// whether the handler has processed any messages
	processedMessages bool
}

func newConsumerHandler(ctx context.Context, name, group string, kotelTracer *kotel.Tracer,
	processorFunc ProcessorFunc, fs FailStrategy, batchSize uint, batchTimeout time.Duration,
	manualCommit, batchMessageDeduplication bool,
) *consumerHandler {
	return &consumerHandler{
		ctx:                       ctx,
		name:                      name,
		group:                     group,
		kotelTracer:               kotelTracer,
		batchSize:                 batchSize,
		batchMessageDeduplication: batchMessageDeduplication,
		ticker:                    time.NewTicker(batchTimeout),
		recBuf:                    make([]*kgo.Record, 0, batchSize),
		mu:                        sync.RWMutex{},
		proc:                      processorFunc,
		failStrategy:              fs,
		manualCommit:              manualCommit,
	}
}

func (c *consumerHandler) consume(ctx context.Context, cl *kgo.Client) error {
	defer c.ticker.Stop()

	for {
		if ctx.Err() != nil {
			if !errors.Is(ctx.Err(), context.Canceled) {
				slog.Info("closing consumer", log.ErrorAttr(ctx.Err()))
			}

			c.mu.Lock()
			err := c.flush(cl)
			c.mu.Unlock()
			if err != nil {
				return err
			}

			return nil
		}

		fetches := cl.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		for _, fetchErr := range fetches.Errors() {
			if !errors.Is(fetchErr.Err, context.Canceled) {
				slog.Error("fetch error", slog.String("topic", fetchErr.Topic),
					slog.Int("partition", int(fetchErr.Partition)), log.ErrorAttr(fetchErr.Err))
			}
		}

		fetches.EachPartition(func(ftp kgo.FetchTopicPartition) {
			for _, rec := range ftp.Records {
				slog.Debug("message claimed", slog.String("value", string(rec.Value)),
					slog.Time("timestamp", rec.Timestamp), slog.String("topic", rec.Topic))
				topicPartitionOffsetDiffGaugeSet(c.ctx, c.group, rec.Topic, rec.Partition, ftp.HighWatermark, rec.Offset)
				messageStatusCountInc(c.ctx, messageReceived, c.group, rec.Topic)

				c.mu.Lock()
				if c.err != nil {
					c.mu.Unlock()
					return
				}

				c.recBuf = append(c.recBuf, rec)
				if uint(len(c.recBuf)) >= c.batchSize {
					err := c.flush(cl)
					if err != nil {
						c.err = err
						c.mu.Unlock()
						return
					}
				}
				c.mu.Unlock()
			}
		})

		if c.getErr() != nil {
			return c.getErr()
		}

		select {
		case <-c.ticker.C:
			c.mu.Lock()
			err := c.flush(cl)
			c.mu.Unlock()
			if err != nil {
				return err
			}
		default:
		}
	}
}

func (c *consumerHandler) flush(cl *kgo.Client) error {
	if len(c.recBuf) == 0 {
		return nil
	}

	recordSpans := make([]recordSpan, 0, len(c.recBuf))
	for _, rec := range c.recBuf {
		ctx, sp := c.getContextWithCorrelation(rec)
		messageStatusCountInc(ctx, messageProcessed, c.group, rec.Topic)
		rec.Context = ctx
		recordSpans = append(recordSpans, recordSpan{record: rec, span: sp})
	}

	records := c.recBuf
	if c.batchMessageDeduplication {
		records = deduplicateRecords(records)
	}

	err := c.proc(c.ctx, records)
	if err != nil {
		if errors.Is(c.ctx.Err(), context.Canceled) {
			c.allowRebalance(cl)
			return fmt.Errorf("context was cancelled after processing error: %w", err)
		}

		err = c.executeFailureStrategy(recordSpans, err)
		if err != nil {
			c.allowRebalance(cl)
			return err
		}
	}

	c.processedMessages = true
	for _, rs := range recordSpans {
		patrontrace.SetSpanSuccess(rs.span)
		rs.span.End()
	}

	if cl != nil {
		if c.manualCommit {
			err = cl.CommitRecords(context.Background(), records...)
			if err != nil {
				cl.AllowRebalance()
				return fmt.Errorf("failed to commit records: %w", err)
			}

			cl.AllowRebalance()
		} else {
			cl.MarkCommitRecords(records...)
		}
	}

	c.recBuf = c.recBuf[:0]

	return nil
}

func (c *consumerHandler) executeFailureStrategy(recordSpans []recordSpan, err error) error {
	switch c.failStrategy {
	case ExitStrategy:
		for _, rs := range recordSpans {
			patrontrace.SetSpanError(rs.span, "executing exit strategy", err)
			rs.span.End()
			messageStatusCountInc(rs.record.Context, messageErrored, c.group, rs.record.Topic)
		}
		slog.Error("could not process message(s)")
		c.err = err
		return err
	case SkipStrategy:
		for _, rs := range recordSpans {
			patrontrace.SetSpanError(rs.span, "executing skip strategy", err)
			rs.span.End()
			messageStatusCountInc(rs.record.Context, messageErrored, c.group, rs.record.Topic)
			messageStatusCountInc(rs.record.Context, messageSkipped, c.group, rs.record.Topic)
		}
		slog.Error("could not process message(s) so skipping with error", log.ErrorAttr(err))
	default:
		slog.Error("unknown failure strategy executed")
		return fmt.Errorf("unknown failure strategy: %v", c.failStrategy)
	}

	return nil
}

func (c *consumerHandler) getContextWithCorrelation(rec *kgo.Record) (context.Context, trace.Span) {
	corID := getCorrelationID(rec.Headers)

	// Use kotel's WithProcessSpan which creates a "{topic} process" span with
	// rich semantic convention attributes. The rec.Context already has trace
	// context extracted by kotel's OnFetchRecordBuffered hook.
	ctx, sp := c.kotelTracer.WithProcessSpan(rec)

	ctx = correlation.ContextWithID(ctx, corID)
	ctx = log.WithContext(ctx, slog.With(slog.String(correlation.ID, corID)))
	return ctx, sp
}

func (c *consumerHandler) allowRebalance(cl *kgo.Client) {
	if c.manualCommit && cl != nil {
		cl.AllowRebalance()
	}
}

func (c *consumerHandler) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.err = err
	}
}

func (c *consumerHandler) getErr() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func getCorrelationID(hh []kgo.RecordHeader) string {
	for _, h := range hh {
		if h.Key == correlation.HeaderID {
			if len(h.Value) > 0 {
				return string(h.Value)
			}
			break
		}
	}

	slog.Debug("correlation header not found, creating new correlation UUID")
	return uuid.New().String()
}

// deduplicateRecords takes a slice of records and de-duplicates based on the Key of those records.
// This function assumes that records are ordered from old to new, and relies on Kafka ordering guarantees within
// partitions. This is the default behaviour from Kafka unless the Producer altered the partition hashing behaviour in
// a nondeterministic way.
func deduplicateRecords(records []*kgo.Record) []*kgo.Record {
	latest := make(map[string]*kgo.Record, len(records))
	for _, rec := range records {
		latest[string(rec.Key)] = rec
	}

	deduplicated := make([]*kgo.Record, 0, len(latest))
	for _, rec := range records {
		if latest[string(rec.Key)] == rec {
			deduplicated = append(deduplicated, rec)
		}
	}

	return deduplicated
}
