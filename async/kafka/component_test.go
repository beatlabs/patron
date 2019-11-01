package kafka

import (
	"bytes"
	"context"
	"encoding/binary"
	go_json "encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
	"time"
)

type testingData struct {
	counter                counter
	msgs                   []*sarama.ConsumerMessage
	decoder                encoding.DecodeRawFunc
	dmsgs                  [][]string
	combinedDecoderVersion int32
}

type counter struct {
	messageCount int
	decodingErr  int
	resultErr    int
	claimErr     int
}

func Test_DecodingMessage(t *testing.T) {

	testingdata := []testingData{
		// expect a decoding error , as we are injecting an erroring decoder implementation
		{
			counter: counter{
				decodingErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
			},
			decoder: erroringDecoder,
		},
		// the json decoder is not compatible with the message raw string format
		{
			counter: counter{
				decodingErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("value", &sarama.RecordHeader{}),
			},
			decoder: go_json.Unmarshal,
		},
		// correctly set up value for the jsonDecoder
		{
			counter: counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{}),
			},
			dmsgs:   [][]string{{"value", "key"}},
			decoder: go_json.Unmarshal,
		},
		// verify positive use of the json as a default decoder
		{
			counter: counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{}),
			},
			dmsgs: [][]string{{"value", "key"}},
		},
		// verify negative use of the json as a default decoder
		{
			counter: counter{
				decodingErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("key value", &sarama.RecordHeader{}),
			},
		},
		// use of the jsonDecoder with multiple messages
		{
			counter: counter{
				decodingErr:  1,
				messageCount: 2,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"key\"]", &sarama.RecordHeader{}),
				saramaConsumerMessage("wrong json", &sarama.RecordHeader{}),
				saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
			},
			dmsgs:   [][]string{{"key"}, {"value"}},
			decoder: go_json.Unmarshal,
		},
		// correctly set up content type for the hardcoded json decoder with message header contentType
		{
			counter: counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte(json.Type),
				}),
			},
			dmsgs: [][]string{{"value", "key"}},
		},
		// correctly set up custom string decoder
		{
			counter: counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("key value", &sarama.RecordHeader{}),
			},
			dmsgs:   [][]string{{"key", "value"}},
			decoder: stringToSliceDecoder,
		},
		// exotic decoder implementation as an example of consuming messages with different schema
		// the approach is an avro like one, where the first byte will point to the right decoder implementation
		{
			counter: counter{
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
		},
	}

	for _, testdata := range testingdata {
		testMessageClaim(t, testdata)
	}

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

func testMessageClaim(t *testing.T, data testingData) {

	ctx := context.Background()

	counter := counter{}

	factory, err := NewComponentBuilder(context.Background(), "name", "topic", "0.0.0.0:9092").
		SetValueDecoder(data.decoder).
		New()

	assert.NoError(t, err, "Could not create factory")

	c, err := factory.Create()

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

func VoidDecoder(data []byte, v interface{}) error {
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
		return go_json.Unmarshal(data[1:], v)
	case 2:
		return stringToSliceDecoder(data[1:], v)
	case 9:
		return erroringDecoder(data[1:], v)
	default:
		return VoidDecoder(data[1:], v)
	}
}

var process = func(counter *counter, data *testingData) func(message async.Message) error {
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
