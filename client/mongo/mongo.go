// Package mongo provides a client implementation for mongo with tracing and metrics included.
package mongo

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const component = "mongo-client"

var cmdDurationMetrics *prometheus.HistogramVec

func init() {
	cmdDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "mongo",
			Name:      "cmd_duration_seconds",
			Help:      "Mongo commands completed by the client.",
		},
		[]string{"command", "success"},
	)
	prometheus.MustRegister(cmdDurationMetrics)
}

// Connect with integrated observability via MongoDB's event package.
func Connect(ctx context.Context, oo ...*options.ClientOptions) (*mongo.Client, error) {
	return mongo.Connect(ctx, append(oo, monitorOption())...)
}

func monitorOption() *options.ClientOptions {
	mon := monitor{
		spans: make(map[key]opentracing.Span),
	}
	return &options.ClientOptions{
		Monitor: &event.CommandMonitor{
			Started:   mon.started,
			Succeeded: mon.succeeded,
			Failed:    mon.failed,
		},
	}
}

type key struct {
	ConnectionID string
	RequestID    int64
}

type monitor struct {
	sync.Mutex
	spans map[key]opentracing.Span
}

func (m *monitor) started(ctx context.Context, startedEvent *event.CommandStartedEvent) {
	sp, _ := trace.ChildSpan(ctx, startedEvent.CommandName, component, ext.SpanKindRPCClient)
	key := createKey(startedEvent.ConnectionID, startedEvent.RequestID)
	m.Lock()
	m.spans[key] = sp
	m.Unlock()
}

func (m *monitor) succeeded(_ context.Context, succeededEvent *event.CommandSucceededEvent) {
	key := createKey(succeededEvent.ConnectionID, succeededEvent.RequestID)
	m.finish(key, succeededEvent.CommandName, true, time.Duration(succeededEvent.DurationNanos))
}

func (m *monitor) failed(_ context.Context, failedEvent *event.CommandFailedEvent) {
	key := createKey(failedEvent.ConnectionID, failedEvent.RequestID)
	m.finish(key, failedEvent.CommandName, false, time.Duration(failedEvent.DurationNanos))
}

func (m *monitor) finish(key key, cmdName string, success bool, duration time.Duration) {
	m.Lock()
	sp, ok := m.spans[key]
	if ok {
		delete(m.spans, key)
	}
	m.Unlock()
	if !ok {
		return
	}
	if success {
		trace.SpanSuccess(sp)
	} else {
		trace.SpanError(sp)
	}

	cmdDurationMetrics.WithLabelValues(cmdName, strconv.FormatBool(success)).Observe(duration.Seconds())
}

func createKey(connID string, reqID int64) key {
	return key{ConnectionID: connID, RequestID: reqID}
}
