package kafka

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	patron_json "github.com/beatlabs/patron/encoding/json"
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
			name:    "does not fail with missing group",
			args:    args{name: "test", brokers: brokers, topic: "topic1", group: ""},
			wantErr: false,
		},
		{
			name:    "success",
			args:    args{name: "test", brokers: brokers, topic: "topic1", group: "group1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.topic, tt.args.group, tt.args.brokers, tt.args.options...)
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
		dec:  patron_json.DecodeRaw,
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

	tests := []struct {
		name    string
		msgs    []*sarama.ConsumerMessage
		error   string
		wantErr bool
	}{
		{"success", saramaConsumerMessages(patron_json.Type), "", false},
		{"failure decoding", saramaConsumerMessages("mock"), "failed to determine decoder for mock", true},
		{"failure content", saramaConsumerMessages(""), "failed to determine content type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chMsg := make(chan async.Message, 1)
			h := handler{messages: chMsg, consumer: &consumer{}}

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

func saramaConsumerMessages(ct string) []*sarama.ConsumerMessage {
	return []*sarama.ConsumerMessage{
		saramaConsumerMessage("value", &sarama.RecordHeader{
			Key:   []byte(encoding.ContentTypeHeader),
			Value: []byte(ct),
		}),
	}
}

func TestConsumer_ConsumeFailedBroker(t *testing.T) {
	f, err := New("name", "topic", "group", []string{"1", "2"})
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	chMsg, chErr, err := c.Consume(context.Background())
	assert.Nil(t, chMsg)
	assert.Nil(t, chErr)
	assert.Error(t, err)
}

func TestConsumer_ConsumeWithGroup(t *testing.T) {
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

	f, err := New("name", "TOPIC", "group", []string{broker.Addr()})
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

func TestConsumer_ConsumeWithoutGroup(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	topic := "foo_topic"
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetVersion(1).
			SetOffset(topic, 0, sarama.OffsetNewest, 10).
			SetOffset(topic, 0, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(topic, 0, 9, sarama.StringEncoder("Foo")),
	})

	f, err := New("name", topic, "", []string{broker.Addr()})
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

type decodingTestData struct {
	counter                eventCounter
	msgs                   []*sarama.ConsumerMessage
	decoder                encoding.DecodeRawFunc
	dmsgs                  [][]string
	combinedDecoderVersion int32
}

type eventCounter struct {
	messageCount int
	decodingErr  int
	resultErr    int
	claimErr     int
}

func TestNilDecoderError(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			decodingErr: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
		},
		decoder: nil,
	}

	testMessageClaim(t, testData)
}

func TestDecodingError(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			decodingErr: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
		},
		decoder: erroringDecoder,
	}

	testMessageClaim(t, testData)
}

func TestIncompatibleDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			decodingErr: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("value", &sarama.RecordHeader{}),
		},
		decoder: json.Unmarshal,
	}

	testMessageClaim(t, testData)
}

func TestJsonDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			messageCount: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{}),
		},
		dmsgs:   [][]string{{"value", "key"}},
		decoder: json.Unmarshal,
	}

	testMessageClaim(t, testData)
}

func TestNoDecoderNoContentType(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			claimErr: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{}),
		},
		dmsgs: [][]string{{"value", "key"}},
	}

	testMessageClaim(t, testData)
}

func TestMultipleMessagesJsonDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			decodingErr:  1,
			messageCount: 2,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"key\"]", &sarama.RecordHeader{}),
			saramaConsumerMessage("wrong json", &sarama.RecordHeader{}),
			saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
		},
		dmsgs:   [][]string{{"key"}, {"value"}},
		decoder: json.Unmarshal,
	}

	testMessageClaim(t, testData)
}

func TestDefaultDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			messageCount: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{
				Key:   []byte(encoding.ContentTypeHeader),
				Value: []byte(patron_json.Type),
			}),
		},
		dmsgs: [][]string{{"value", "key"}},
	}

	testMessageClaim(t, testData)
}

func TestStringDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			messageCount: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("key value", &sarama.RecordHeader{}),
		},
		dmsgs:   [][]string{{"key", "value"}},
		decoder: stringToSliceDecoder,
	}

	testMessageClaim(t, testData)
}

func TestExoticDecoder(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			messageCount: 3,
			resultErr:    1,
			decodingErr:  2,
		},
		msgs: []*sarama.ConsumerMessage{
			// will use json decoder based on the message header but fail due to bad json
			versionedConsumerMessage("\"key\" \"value\"]", &sarama.RecordHeader{}, 1),
			// will use json decoder based on the message header
			versionedConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{}, 1),
			// will fail at the result level due to the wrong message header, string instead of json
			versionedConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{}, 2),
			// will use void decoder because there is no message header
			versionedConsumerMessage("any string ... ", &sarama.RecordHeader{}, 99),
			// will produce error due to the content type invoking the erroringDecoder
			versionedConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{}, 9),
			// will use string decoder based on the message header
			versionedConsumerMessage("key value", &sarama.RecordHeader{}, 2),
		},
		dmsgs:   [][]string{{"key", "value"}, {}, {"key", "value"}},
		decoder: combinedDecoder,
	}

	testMessageClaim(t, testData)
}

func saramaConsumerMessage(value string, header *sarama.RecordHeader) *sarama.ConsumerMessage {
	return versionedConsumerMessage(value, header, 0)
}

func versionedConsumerMessage(value string, header *sarama.RecordHeader, version uint8) *sarama.ConsumerMessage {

	bytes := []byte(value)

	if version > 0 {
		bytes = append([]byte{version}, bytes...)
	}

	return &sarama.ConsumerMessage{
		Topic:          "TEST_TOPIC",
		Partition:      0,
		Key:            []byte("key"),
		Value:          bytes,
		Offset:         0,
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Headers:        []*sarama.RecordHeader{header},
	}
}

func testMessageClaim(t *testing.T, data decodingTestData) {

	ctx := context.Background()

	counter := eventCounter{}

	factory, err := New("name", "topic", "group", []string{"0.0.0.0:9092"}, Decoder(data.decoder))

	assert.NoError(t, err, "Could not create factory")

	c, err := factory.Create()

	if data.decoder == nil {
		assert.Error(t, err)
		return
	}

	assert.NoError(t, err, "Could not create component")

	// do a dirty cast for the sake of facilitating the test
	kc := reflect.ValueOf(c).Elem().Interface().(consumer)

	// claim and process the messages and update the counters accordingly
	for _, km := range data.msgs {

		if data.combinedDecoderVersion != 0 {
			km.Value = append([]byte{byte(data.combinedDecoderVersion)}, km.Value...)

		}

		msg, err := claimMessage(ctx, &kc, km)

		if err != nil {
			counter.claimErr++
			continue
		}

		err = process(&counter, &data)(msg)
		if err != nil {
			println(fmt.Sprintf("Could not process message %v : %v", msg, err))
		}
	}

	assert.Equal(t, data.counter, counter)

}

// some naive decoder implementations for testing

func erroringDecoder(data []byte, v interface{}) error {
	return fmt.Errorf("Predefined Decoder Error for message %s", string(data))
}

func voidDecoder(data []byte, v interface{}) error {
	return nil
}

func stringToSliceDecoder(data []byte, v interface{}) error {
	if arr, ok := v.(*[]string); ok {
		*arr = append(*arr, strings.Split(string(data), " ")...)
	} else {
		return fmt.Errorf("Provided object is not valid for splitting data into a slice '%v'", v)
	}
	return nil
}

func combinedDecoder(data []byte, v interface{}) error {

	version, _ := binary.ReadUvarint(bytes.NewBuffer(data[:1]))

	switch version {
	case 1:
		return json.Unmarshal(data[1:], v)
	case 2:
		return stringToSliceDecoder(data[1:], v)
	case 9:
		return erroringDecoder(data[1:], v)
	default:
		return voidDecoder(data[1:], v)
	}
}

var process = func(counter *eventCounter, data *decodingTestData) func(message async.Message) error {
	return func(message async.Message) error {
		// we always assume we will decode to a slice of strings
		values := []string{}
		// we assume based on our transform function, that we will be able to decode as a rule
		if err := message.Decode(&values); err != nil {
			counter.decodingErr++
			return fmt.Errorf("Error encountered while decoding message from source [%v] : %v", message, err)
		}
		if !reflect.DeepEqual(data.dmsgs[counter.messageCount], values) {
			counter.resultErr++
			return fmt.Errorf("Could not verify equality for '%v' and '%v' at index '%d'", values, data.dmsgs[counter.messageCount], counter.messageCount)
		}
		counter.messageCount++
		return nil
	}
}
