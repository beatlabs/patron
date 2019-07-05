package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/async"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	brokers := []string{"192.168.1.1"}
	type args struct {
		name    string
		brokers []string
		topic   string
		group   string
		options []OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "fails with missing name",
			args:    args{name: "", brokers: brokers, topic: "topic1", group: "group1"},
			wantErr: true,
		},
		{
			name:    "fails with missing brokers",
			args:    args{name: "test", brokers: []string{}, topic: "topic1", group: "group1"},
			wantErr: true,
		},
		{
			name:    "fails with missing topics",
			args:    args{name: "test", brokers: brokers, topic: "", group: "group1"},
			wantErr: true,
		},
		{
			name:    "fails with missing group",
			args:    args{name: "test", brokers: brokers, topic: "topic1", group: ""},
			wantErr: true,
		},
		{
			name:    "success",
			args:    args{name: "test", brokers: brokers, topic: "topic1", group: "group1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, "", tt.args.topic, tt.args.group, tt.args.brokers, tt.args.options...)
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

func TestFactory_Create(t *testing.T) {
	type fields struct {
		oo []OptionFunc
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "success", wantErr: false},
		{name: "failed with invalid option", fields: fields{oo: []OptionFunc{Buffer(-100)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{
				name:    "test",
				ct:      "",
				topic:   "topic",
				brokers: []string{"192.168.1.1"},
				oo:      tt.fields.oo,
			}
			got, err := f.Create()
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

func Test_determineContentType(t *testing.T) {
	type args struct {
		hdr []*sarama.RecordHeader
	}

	validHdr := &sarama.RecordHeader{
		Key:   []byte(encoding.ContentTypeHeader),
		Value: []byte("val1"),
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"failure", args{hdr: []*sarama.RecordHeader{}}, "", true},
		{"success", args{hdr: []*sarama.RecordHeader{validHdr}}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineContentType(tt.args.hdr)
			if tt.wantErr {
				assert.Empty(t, got)
				assert.Error(t, err)
			} else {
				assert.NotNil(t, got)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_message(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	sp := opentracing.StartSpan("test")
	ctx := context.Background()
	cm := &sarama.ConsumerMessage{
		Value: []byte(`{"key":"value"}`),
	}
	sess := &mockConsumerSession{}
	msg := message{
		sess: sess,
		ctx:  ctx,
		dec:  json.DecodeRaw,
		span: sp,
		msg:  cm,
	}

	assert.NotNil(t, msg.Context())
	assert.NoError(t, msg.Ack())
	assert.NoError(t, msg.Nack())
	m := make(map[string]string)
	assert.NoError(t, msg.Decode(&m))
	assert.Equal(t, "value", m["key"])
}

func TestMapHeader(t *testing.T) {
	hh := []*sarama.RecordHeader{
		&sarama.RecordHeader{
			Key:   []byte("key"),
			Value: []byte("value"),
		},
	}
	hdr := mapHeader(hh)
	assert.Equal(t, "value", hdr["key"])
}

type mockConsumerClaim struct{ msgs []*sarama.ConsumerMessage }

func (m *mockConsumerClaim) Messages() <-chan *sarama.ConsumerMessage {
	ch := make(chan *sarama.ConsumerMessage, len(m.msgs))
	for _, m := range m.msgs {
		ch <- m
	}
	go func() {
		close(ch)
	}()
	return ch
}
func (m *mockConsumerClaim) Topic() string              { return "" }
func (m *mockConsumerClaim) Partition() int32           { return 0 }
func (m *mockConsumerClaim) InitialOffset() int64       { return 0 }
func (m *mockConsumerClaim) HighWaterMarkOffset() int64 { return 1 }

type mockConsumerSession struct{}

func (m *mockConsumerSession) Claims() map[string][]int32 { return nil }
func (m *mockConsumerSession) MemberID() string           { return "" }
func (m *mockConsumerSession) GenerationID() int32        { return 0 }
func (m *mockConsumerSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *mockConsumerSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}
func (m *mockConsumerSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {}
func (m *mockConsumerSession) Context() context.Context                                 { return nil }

func TestHandler_ConsumeClaim(t *testing.T) {
	msgs := []*sarama.ConsumerMessage{
		{
			Topic:          "TEST_TOPIC",
			Partition:      0,
			Key:            []byte("key"),
			Value:          []byte("value"),
			Offset:         0,
			Timestamp:      time.Now(),
			BlockTimestamp: time.Now(),
		},
	}

	tests := []struct {
		name        string
		msgs        []*sarama.ConsumerMessage
		contentType string
		error       string
		wantErr     bool
	}{
		{"success", msgs, json.Type, "", false},
		{"failure decoding", msgs, "mock", "failed to determine decoder for mock", true},
		{"failure content", msgs, "", "failed to determine content type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chMsg := make(chan async.Message, 1)
			h := handler{messages: chMsg, consumer: &consumer{contentType: tt.contentType}}
			err := h.ConsumeClaim(&mockConsumerSession{}, &mockConsumerClaim{tt.msgs})

			if tt.wantErr {
				assert.Error(t, err, tt.error)
			} else {
				assert.NoError(t, err)
				ch := <-chMsg
				assert.NotNil(t, ch)
			}
		})
	}
}

func TestConsumer_ConsumeFailedBroker(t *testing.T) {
	f, err := New("name", "application/json", "topic", "group", []string{"1", "2"})
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	chMsg, chErr, err := c.Consume(context.Background())
	assert.Nil(t, chMsg)
	assert.Nil(t, chErr)
	assert.Error(t, err)
}

func TestConsumer_Consume(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader("TOPIC", 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset("TOPIC", 0, sarama.OffsetNewest, 10).
			SetOffset("TOPIC", 0, sarama.OffsetOldest, 7),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage("TOPIC", 0, 9, sarama.StringEncoder("Foo")).
			SetHighWaterMark("TOPIC", 0, 14),
	})

	f, err := New("name", "application/json", "TOPIC", "group", []string{broker.Addr()})
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	ctx := context.Background()
	chMsg, chErr, err := c.Consume(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, chMsg)
	assert.NotNil(t, chErr)

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()

	ctx.Done()
}
