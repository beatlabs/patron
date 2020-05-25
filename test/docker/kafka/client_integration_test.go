// +build integration

package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	client "github.com/beatlabs/patron/client/kafka"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	clientTopic = "clientTopic"
)

func TestNewAsyncProducer_Success(t *testing.T) {
	ap, chErr, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
}

func TestNewSyncProducer_Success(t *testing.T) {
	p, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	msg := client.NewMessage(clientTopic, "TEST")
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	ap, chErr, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	err = ap.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, ap.Close())
	assert.Len(t, mtr.FinishedSpans(), 1)

	expected := map[string]interface{}{
		"component": "kafka-async-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "async",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	msg := client.NewMessage(clientTopic, "TEST")
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	p, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	require.NoError(t, err)
	assert.NotNil(t, p)
	err = p.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, p.Close())
	assert.Len(t, mtr.FinishedSpans(), 1)

	expected := map[string]interface{}{
		"component": "kafka-sync-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "sync",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func TestAsyncProducer_SendMessage_WithKey(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	testKey := "TEST"
	msg, err := client.NewMessageWithKey(clientTopic, "TEST", testKey)
	assert.NoError(t, err)
	ap, chErr, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NoError(t, err)
	err = ap.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, ap.Close())
	expected := map[string]interface{}{
		"component": "kafka-async-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "async",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func TestSyncProducer_SendMessage_WithKey(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	testKey := "TEST"
	msg, err := client.NewMessageWithKey(clientTopic, "TEST", testKey)
	assert.NoError(t, err)
	ap, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	require.NoError(t, err)
	assert.NotNil(t, ap)
	err = ap.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, ap.Close())
	expected := map[string]interface{}{
		"component": "kafka-sync-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "sync",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func TestAsyncProducerActiveBrokers(t *testing.T) {
	ap, chErr, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSyncProducerActiveBrokers(t *testing.T) {
	ap, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSendWithCustomEncoder(t *testing.T) {
	var u examples.User
	firstName, lastName := "John", "Doe"
	u.Firstname = &firstName
	u.Lastname = &lastName
	tests := map[string]struct {
		data        interface{}
		key         string
		enc         encoding.EncodeFunc
		ct          string
		wantSendErr bool
	}{
		"json success":                {data: "testdata1", key: "testkey1", enc: json.Encode, ct: json.Type, wantSendErr: false},
		"protobuf success":            {data: &u, key: "testkey2", enc: protobuf.Encode, ct: protobuf.Type, wantSendErr: false},
		"failure due to invalid data": {data: make(chan bool), key: "testkey3", wantSendErr: true},
		"nil message data":            {data: nil, key: "testkey4", wantSendErr: false},
		"nil encoder":                 {data: "somedata", key: "testkey5", ct: json.Type, wantSendErr: false},
		"empty data":                  {data: "", key: "testkey6", enc: json.Encode, ct: json.Type, wantSendErr: false},
		"empty data two":              {data: "", key: "🚖", enc: json.Encode, ct: json.Type, wantSendErr: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			msg, _ := client.NewMessageWithKey("TOPIC", tt.data, tt.key)

			ap, err := client.NewBuilder(Brokers()).WithVersion(sarama.V2_1_0_0.String()).WithEncoder(tt.enc, tt.ct).CreateSync()
			if tt.enc != nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, ap)
			err = ap.Send(context.Background(), msg)
			if tt.wantSendErr == false {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func Test_createAsyncProducerUsingBuilder(t *testing.T) {

	var builderNoErrors []error
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
		ack         client.RequiredAcks
		timeout     time.Duration
		enc         encoding.EncodeFunc
		contentType string
		wantErrs    []error
	}{
		"success": {
			brokers:     Brokers(),
			version:     sarama.V2_1_0_0.String(),
			ack:         client.NoResponse,
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
			got, chErr, gotErrs := client.NewBuilder(tt.brokers).
				WithVersion(tt.version).
				WithRequiredAcksPolicy(tt.ack).
				WithTimeout(tt.timeout).
				WithEncoder(tt.enc, tt.contentType).
				CreateAsync()

			if len(tt.wantErrs) > 0 {
				assert.ObjectsAreEqual(tt.wantErrs, gotErrs)
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.NotNil(t, chErr)
				assert.IsType(t, &client.AsyncProducer{}, got)
			}
		})
	}

}
