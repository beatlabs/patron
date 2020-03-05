package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/trace"
	"github.com/beatlabs/patron/trace/kafka"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go"
)

type testMetric struct {
	metric *prometheus.CounterVec
	name   string
	label  string
	count  uint64
}

func TestNewMessage(t *testing.T) {
	m := kafka.NewMessage("TOPIC", []byte("TEST"))
	assert.Equal(t, "TOPIC", m.Topic)
	assert.Equal(t, []byte("TEST"), m.Body)
}

func TestNewMessageWithKey(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		key     string
		wantErr bool
	}{
		{name: "success", data: []byte("TEST"), key: "TEST"},
		{name: "failure due to empty message key", data: []byte("TEST"), key: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kafka.NewMessageWithKey("TOPIC", tt.data, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewSyncProducer_Failure(t *testing.T) {
	ab := AsyncBuilder{kafka.NewBuilder([]string{})}
	got, err := ab.Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	ab := AsyncBuilder{kafka.NewBuilder([]string{"xxx"}).WithVersion("xxxx")}
	got, err := ab.Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Success(t *testing.T) {
	seed := createKafkaBroker(t, false)
	ab := AsyncBuilder{kafka.NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String())}
	got, err := ab.Create()
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	msg := kafka.NewMessage("TOPIC", "TEST")
	tm := testMetric{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}
	seed := createKafkaBroker(t, true)
	sb := AsyncBuilder{kafka.NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String())}
	ap, err := sb.Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = ap.Send(ctx, msg)
	assertMetric(t, tm)
	assert.NoError(t, err)
	assert.Error(t, <-ap.Error())
	assert.NoError(t, ap.Close())
}

func TestAsyncProducer_SendMessage_WithKey(t *testing.T) {
	testKey := "TEST"
	msg, err := kafka.NewMessageWithKey("TOPIC", "TEST", testKey)
	tm := testMetric{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}
	assert.Equal(t, testKey, *msg.Key)
	assert.NoError(t, err)
	seed := createKafkaBroker(t, true)
	ab := AsyncBuilder{kafka.NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String())}
	ap, err := ab.Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	clearMetrics(tm)
	err = ap.Send(ctx, msg)
	assertMetric(t, tm)
	assert.NoError(t, err)
	assert.Error(t, <-ap.Error())
	assert.NoError(t, ap.Close())
}

func createKafkaBroker(t *testing.T, retError bool) *sarama.MockBroker {
	lead := sarama.NewMockBroker(t, 2)
	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(lead.Addr(), lead.BrokerID())
	metadataResponse.AddTopicPartition("TOPIC", 0, lead.BrokerID(), nil, nil, sarama.ErrNoError)

	prodSuccess := new(sarama.ProduceResponse)
	if retError {
		prodSuccess.AddTopicPartition("TOPIC", 0, sarama.ErrDuplicateSequenceNumber)
	} else {
		prodSuccess.AddTopicPartition("TOPIC", 0, sarama.ErrNoError)
	}
	lead.Returns(prodSuccess)

	config := sarama.NewConfig()
	config.Producer.Flush.Messages = 10
	config.Producer.Return.Successes = true
	seed := sarama.NewMockBroker(t, 1)
	seed.Returns(metadataResponse)
	return seed
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
		{name: "json success", data: "testdata1", key: "testkey1", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}}, wantSendErr: false},
		{name: "protobuf success", data: &u, key: "testkey2", enc: protobuf.Encode, ct: protobuf.Type, tm: []testMetric{{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}}, wantSendErr: false},
		{name: "failure due to invalid data", data: make(chan bool), key: "testkey3", wantSendErr: true},
		{name: "nil message data", data: nil, key: "testkey4", wantSendErr: false},
		{name: "nil encoder", data: "somedata", key: "testkey5", ct: json.Type, wantSendErr: false},
		{name: "empty data", data: "", key: "testkey6", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}}, wantSendErr: false},
		{name: "empty data two", data: "", key: "🚖", enc: json.Encode, ct: json.Type, tm: []testMetric{{messageStatus, "component_kafka_async_producer_message_status", "sent", 1}}, wantSendErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearMetrics(tt.tm...)
			msg, _ := kafka.NewMessageWithKey("TOPIC", tt.data, tt.key)

			seed := createKafkaBroker(t, true)
			ab := AsyncBuilder{kafka.NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String()).WithEncoder(tt.enc, tt.ct)}
			ap, err := ab.Create()
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
	// Loop over our test metrics
	for _, v := range testMetrics {
		// And find the one which matches the label
		for _, mf := range metricFamilies {
			for _, m := range mf.Metric {
				for _, l := range m.Label {
					if *l.Value == v.label {
						current = m
					}
				}
			}
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

func Test_createAsyncProducerUsingBuilder(t *testing.T) {
	seed := createKafkaBroker(t, true)

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
		ack         kafka.RequiredAcks
		timeout     time.Duration
		enc         encoding.EncodeFunc
		contentType string
		wantErrs    []error
	}{
		"success": {
			brokers:     []string{seed.Addr()},
			version:     sarama.V0_8_2_0.String(),
			ack:         kafka.NoResponse,
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
			ab := AsyncBuilder{kafka.NewBuilder(tt.brokers).
				WithVersion(tt.version).
				WithRequiredAcksPolicy(tt.ack).
				WithTimeout(tt.timeout).
				WithEncoder(tt.enc, tt.contentType)}
			gotAsyncProducer, gotErrs := ab.Create()

			v, _ := sarama.ParseKafkaVersion(tt.version)
			if len(tt.wantErrs) > 0 {
				assert.ObjectsAreEqual(tt.wantErrs, gotErrs)
				assert.Nil(t, gotAsyncProducer)
			} else {
				assert.NotNil(t, gotAsyncProducer)
				assert.IsType(t, &AsyncProducer{}, gotAsyncProducer)
				assert.EqualValues(t, v, gotAsyncProducer.cfg.Version)
				assert.EqualValues(t, tt.ack, gotAsyncProducer.cfg.Producer.RequiredAcks)
				assert.Equal(t, tt.timeout, gotAsyncProducer.cfg.Net.DialTimeout)
			}
		})
	}

}
