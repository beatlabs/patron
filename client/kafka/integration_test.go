// +build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"

	"github.com/Shopify/sarama"
	dockerKafka "github.com/beatlabs/patron/test/docker/kafka"
	"github.com/beatlabs/patron/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
)

const (
	topic = "Topic"
)

func TestMain(m *testing.M) {
	os.Exit(dockerKafka.RunWithKafka(m, 120*time.Second, getTopic(topic)))
}

func TestNewAsyncProducer_Success(t *testing.T) {
	ap, chErr, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
}

func TestNewSyncProducer_Success(t *testing.T) {
	p, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	msg := NewMessage(topic, "TEST")
	tm := testMetric{messageStatus, "component_kafka_async_producer_message_status", []string{"sent", "async"}, 1}
	ap, chErr, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = ap.Send(ctx, msg)
	assert.NoError(t, err)
	assertMetric(t, tm)
	assert.NoError(t, ap.Close())
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	msg := NewMessage(topic, "TEST")
	tm := testMetric{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}
	p, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	require.NoError(t, err)
	assert.NotNil(t, p)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = p.Send(ctx, msg)
	assert.NoError(t, err)
	assertMetric(t, tm)
	assert.NoError(t, p.Close())
}

func TestAsyncProducer_SendMessage_WithKey(t *testing.T) {
	testKey := "TEST"
	msg, err := NewMessageWithKey(topic, "TEST", testKey)
	tm := testMetric{messageStatus, "component_kafka_async_producer_message_status", []string{"sent", "async"}, 1}
	assert.Equal(t, testKey, *msg.key)
	assert.NoError(t, err)
	ap, chErr, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = ap.Send(ctx, msg)
	assert.NoError(t, err)
	assertMetric(t, tm)
	assert.NoError(t, ap.Close())
}

func TestSyncProducer_SendMessage_WithKey(t *testing.T) {
	testKey := "TEST"
	msg, err := NewMessageWithKey(topic, "TEST", testKey)
	tm := testMetric{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}
	assert.Equal(t, testKey, *msg.key)
	assert.NoError(t, err)
	ap, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	require.NoError(t, err)
	assert.NotNil(t, ap)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = ap.Send(ctx, msg)
	assert.NoError(t, err)
	assertMetric(t, tm)
	assert.NoError(t, ap.Close())
}

func TestAsyncProducerActiveBrokers(t *testing.T) {
	ap, chErr, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSyncProducerActiveBrokers(t *testing.T) {
	ap, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSendWithCustomEncoder(t *testing.T) {
	var u examples.User
	firstname, lastname := "John", "Doe"
	u.Firstname = &firstname
	u.Lastname = &lastname
	tests := []struct {
		name        string
		data        interface{}
		key         string
		enc         encoding.EncodeFunc
		ct          string
		tm          []testMetric
		wantSendErr bool
	}{
		{name: "json success", data: "testdata1", key: "testkey1", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}}, wantSendErr: false},
		{name: "protobuf success", data: &u, key: "testkey2", enc: protobuf.Encode, ct: protobuf.Type, tm: []testMetric{{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}}, wantSendErr: false},
		{name: "failure due to invalid data", data: make(chan bool), key: "testkey3", wantSendErr: true},
		{name: "nil message data", data: nil, key: "testkey4", wantSendErr: false},
		{name: "nil encoder", data: "somedata", key: "testkey5", ct: json.Type, wantSendErr: false},
		{name: "empty data", data: "", key: "testkey6", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}}, wantSendErr: false},
		{name: "empty data two", data: "", key: "🚖", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_sync_producer_message_status", []string{"sent", "sync"}, 1}}, wantSendErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearMetrics(tt.tm...)
			msg, _ := NewMessageWithKey("TOPIC", tt.data, tt.key)

			ap, err := NewBuilder(dockerKafka.Brokers()).WithVersion(sarama.V2_1_0_0.String()).WithEncoder(tt.enc, tt.ct).CreateSync()
			if tt.enc != nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, ap)
			err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
			assert.NoError(t, err)
			_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
			err = ap.Send(ctx, msg)
			assertMetric(t, tt.tm...)
			if tt.wantSendErr == false {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func Test_createAsyncProducerUsingBuilder(t *testing.T) {

	var builderNoErrors = []error{}
	var builderAllErrors = []error{
		errors.New("brokers list is empty"),
		errors.New("encoder is nil"),
		errors.New("content type is empty"),
		errors.New("dial timeout has to be positive"),
		errors.New("version is required"),
		errors.New("invalid value for required acks policy provided"),
	}

	tests := map[string]struct {
		brokers     []string
		version     string
		ack         RequiredAcks
		timeout     time.Duration
		enc         encoding.EncodeFunc
		contentType string
		wantErrs    []error
	}{
		"success": {
			brokers:     dockerKafka.Brokers(),
			version:     sarama.V2_1_0_0.String(),
			ack:         NoResponse,
			timeout:     1 * time.Second,
			enc:         json.Encode,
			contentType: json.Type,
			wantErrs:    builderNoErrors,
		},
		"error in all builder steps": {
			brokers:     []string{},
			version:     "",
			ack:         -5,
			timeout:     -1 * time.Second,
			enc:         nil,
			contentType: "",
			wantErrs:    builderAllErrors,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotAsyncProducer, chErr, gotErrs := NewBuilder(tt.brokers).
				WithVersion(tt.version).
				WithRequiredAcksPolicy(tt.ack).
				WithTimeout(tt.timeout).
				WithEncoder(tt.enc, tt.contentType).
				CreateAsync()

			v, _ := sarama.ParseKafkaVersion(tt.version)
			if len(tt.wantErrs) > 0 {
				assert.ObjectsAreEqual(tt.wantErrs, gotErrs)
				assert.Nil(t, gotAsyncProducer)
			} else {
				assert.NotNil(t, gotAsyncProducer)
				assert.NotNil(t, chErr)
				assert.IsType(t, &AsyncProducer{}, gotAsyncProducer)
				assert.EqualValues(t, v, gotAsyncProducer.cfg.Version)
				assert.EqualValues(t, tt.ack, gotAsyncProducer.cfg.Producer.RequiredAcks)
				assert.Equal(t, tt.timeout, gotAsyncProducer.cfg.Net.DialTimeout)
			}
		})
	}

}

func getTopic(name string) string {
	return fmt.Sprintf("%s:1:1", name)
}

type testMetric struct {
	metric *prometheus.CounterVec
	name   string
	labels []string
	count  uint64
}

func clearMetrics(testMetrics ...testMetric) {
	for _, v := range testMetrics {
		v.metric.Reset()
	}
}

func assertMetric(t *testing.T, testMetrics ...testMetric) {
	reg := prometheus.NewRegistry()
	for _, v := range testMetrics {
		err := reg.Register(v.metric)
		assert.NoError(t, err)
	}
	metricFamilies, err := reg.Gather()
	assert.NoError(t, err)
	assert.Len(t, metricFamilies, len(testMetrics))

	var current *io_prometheus_client.Metric
	found := map[string]struct{}{}
	expected := map[string]struct{}{}
	// Loop over our test metrics
	for _, v := range testMetrics {
		for _, l := range v.labels {
			expected[l] = struct{}{}
		}

		// And find the one which matches the labels
		for _, mf := range metricFamilies {
			for _, m := range mf.Metric {
				for _, l := range m.Label {
					found[*l.Value] = struct{}{}

					for _, lbl := range v.labels {
						if *l.Value == lbl {
							current = m
							break
						}
					}
				}
			}
		}
		// will fail in case of metric mismatch e.g. `creation-errors` instead of `sent`
		assert.NotNil(t, current)
		if current == nil {
			t.Errorf("found: %v expected: %v", found, expected)
			continue
		}

		// Then, perform the assertions on the matched counter
		counter := current.Counter
		if v.count > 0 {
			assert.NotNil(t, v.metric)
			assert.NotNil(t, counter)
			assert.Equal(t, v.count, uint64(*counter.Value))
		} else {
			assert.Nil(t, v.metric)
			assert.Nil(t, counter)
		}
		counter.Reset()
	}
}
