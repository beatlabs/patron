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
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	patron_json "github.com/beatlabs/patron/encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSaramaConfig(t *testing.T) {
	sc, err := DefaultSaramaConfig("name")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(sc.ClientID, fmt.Sprintf("-%s", "name")))
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
		Topic: "topicone",
		Value: []byte(`{"key":"value"}`),
	}
	msg := message{
		sess: nil,
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
	assert.Equal(t, "topicone", msg.Source())
}

func TestMapHeader(t *testing.T) {
	hh := []*sarama.RecordHeader{
		{
			Key:   []byte("key"),
			Value: []byte("value"),
		},
	}
	hdr := mapHeader(hh)
	assert.Equal(t, "value", hdr["key"])
}

func Test_getCorrelationID(t *testing.T) {
	withID := []*sarama.RecordHeader{{Key: []byte(correlation.HeaderID), Value: []byte("123")}}
	withoutID := []*sarama.RecordHeader{{Key: []byte(correlation.HeaderID), Value: []byte("")}}
	missingHeader := []*sarama.RecordHeader{}
	type args struct {
		hh []*sarama.RecordHeader
	}
	tests := map[string]struct {
		args args
	}{
		"with id":        {args: args{hh: withID}},
		"without id":     {args: args{hh: withoutID}},
		"missing header": {args: args{hh: missingHeader}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, getCorrelationID(tt.args.hh))
		})
	}
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

func TestWrongContentTypeError(t *testing.T) {

	testData := decodingTestData{
		counter: eventCounter{
			claimErr: 1,
		},
		msgs: []*sarama.ConsumerMessage{
			saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{Key: []byte(encoding.ContentTypeHeader), Value: []byte(`something`)}),
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

	// claim and process the messages and update the counters accordingly
	for _, km := range data.msgs {

		if data.combinedDecoderVersion != 0 {
			km.Value = append([]byte{byte(data.combinedDecoderVersion)}, km.Value...)

		}

		msg, err := ClaimMessage(ctx, km, data.decoder, nil)

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
