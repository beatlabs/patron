package kafka

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/trace"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go"
)

func TestNewMessage(t *testing.T) {
	m := NewMessage("TOPIC", []byte("TEST"))
	assert.Equal(t, "TOPIC", m.topic)
	assert.Equal(t, []byte("TEST"), m.body)
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
			got, err := NewMessageWithKey("TOPIC", tt.data, tt.key)
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
	got, err := NewBuilder([]string{}).Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	got, err := NewBuilder([]string{"xxx"}).WithVersion("xxxx").Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Success(t *testing.T) {
	seed := createKafkaBroker(t, false)
	got, err := NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String()).Create()
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	msg := NewMessage("TOPIC", "TEST")
	seed := createKafkaBroker(t, true)
	ap, err := NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String()).Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	err = ap.Send(ctx, msg)
	assert.NoError(t, err)
	assert.Error(t, <-ap.Error())
	assert.NoError(t, ap.Close())
}

func TestAsyncProducer_SendMessage_WithKey(t *testing.T) {
	testKey := "TEST"
	msg, err := NewMessageWithKey("TOPIC", "TEST", testKey)
	assert.Equal(t, testKey, *msg.key)
	assert.NoError(t, err)
	seed := createKafkaBroker(t, true)
	ap, err := NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String()).Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	err = trace.Setup("test", "1.0.0", "0.0.0.0:6831", jaeger.SamplerTypeProbabilistic, 0.1)
	assert.NoError(t, err)
	_, ctx := trace.ChildSpan(context.Background(), "123", "cmp")
	err = ap.Send(ctx, msg)
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
		wantSendErr bool
	}{
		{name: "json success", data: "testdata1", key: "testkey1", enc: json.Encode, ct: json.Type, wantSendErr: false},
		{name: "protobuf success", data: &u, key: "testkey2", enc: protobuf.Encode, ct: protobuf.Type, wantSendErr: false},
		{name: "failure due to invalid data", data: make(chan bool), key: "testkey3", wantSendErr: true},
		{name: "nil message data", data: nil, key: "testkey4", wantSendErr: false},
		{name: "nil encoder", data: "somedata", key: "testkey5", ct: json.Type, wantSendErr: false},
		{name: "empty data", data: "", key: "testkey6", enc: json.Encode, ct: json.Type, wantSendErr: false},
		{name: "empty data two", data: "", key: "🚖", enc: json.Encode, ct: json.Type, wantSendErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, _ := NewMessageWithKey("TOPIC", tt.data, tt.key)

			seed := createKafkaBroker(t, true)
			ap, err := NewBuilder([]string{seed.Addr()}).WithVersion(sarama.V0_8_2_0.String()).WithEncoder(tt.enc, tt.ct).Create()
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
			if tt.wantSendErr == false {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
