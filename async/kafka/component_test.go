package kafka

import (
	"context"
	go_json "encoding/json"
	"errors"
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
	counter             Counter
	msgs                []*sarama.ConsumerMessage
	decoder             func(contentType string) (encoding.DecodeRawFunc, error)
	consumerContentType string
	dmsgs               [][]string
}

type Counter struct {
	messageCount int
	decodingErr  int
	resultErr    int
	claimErr     int
}

func Test_DecodingMessage(t *testing.T) {

	testingdata := []testingData{
		// expect a decoding error , as we are injecting an erroring decoder implementation
		{
			counter: Counter{
				decodingErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
			},
			decoder: erroringDecoder,
		},
		// we expect one error during the claimMessage step ,
		// because our test jsonDecoder expects a contentType in the header or the consumer
		{
			counter: Counter{
				claimErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{}),
			},
			decoder: jsonDecoder,
		},
		// correctly set up content type for the test jsonDecoder with message header
		{
			counter: Counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("json"),
				}),
			},
			dmsgs:   [][]string{{"value"}},
			decoder: jsonDecoder,
		},
		// correctly set up content type for the test jsonDecoder with consumer contentType
		{
			counter: Counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{}),
			},
			dmsgs:               [][]string{{"value", "key"}},
			decoder:             jsonDecoder,
			consumerContentType: "json",
		},
		// wrongly set up content type for the hardcoded json decoder
		// with consumer contentType overriding message header contentType
		{
			counter: Counter{
				claimErr: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte(json.Type),
				}),
			},
			dmsgs:               [][]string{{"value", "key"}},
			consumerContentType: "json",
		},
		// correctly set up content type for the hardcoded json decoder
		// with consumer contentType overriding message header contentType
		{
			counter: Counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("[\"value\",\"key\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("json"),
				}),
			},
			dmsgs:               [][]string{{"value", "key"}},
			consumerContentType: json.Type,
		},
		// correctly set up content type for the hardcoded json decoder with message header contentType
		{
			counter: Counter{
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
		// correctly set up custom decoder
		{
			counter: Counter{
				messageCount: 1,
			},
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("key value", &sarama.RecordHeader{}),
			},
			dmsgs:   [][]string{{"key", "value"}},
			decoder: stringToSliceDecoder,
		},
		{
			counter: Counter{
				messageCount: 3,
				resultErr:    1,
				decodingErr:  2,
			},
			msgs: []*sarama.ConsumerMessage{
				// will use json decoder based on the message header but fail due to bad json
				saramaConsumerMessage("\"key\" \"value\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("json"),
				}),
				// will use json decoder based on the message header
				saramaConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("json"),
				}),
				// will fail at the result level due to the wrong message header, string instead of json
				saramaConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("string"),
				}),
				// will use void decoder because there is no message header
				saramaConsumerMessage("any string ... ", &sarama.RecordHeader{}),
				// will produce error due to the content type invoking the erroringDecoder
				saramaConsumerMessage("[\"key\",\"value\"]", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("error"),
				}),
				// will use string decoder based on the message header
				saramaConsumerMessage("key value", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte("string"),
				}),
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
	return &sarama.ConsumerMessage{
		Topic:          "TEST_TOPIC",
		Partition:      0,
		Key:            []byte("key"),
		Value:          []byte(value),
		Offset:         0,
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Headers:        []*sarama.RecordHeader{header},
	}
}

func testMessageClaim(t *testing.T, data testingData) {

	ctx := context.Background()

	counter := Counter{}

	factory, err := NewComponentBuilder(context.Background(), "name", "topic", "0.0.0.0:9092").
		SetValueDecoder(data.decoder).
		SetContentType(data.consumerContentType).
		New()

	if err != nil {
		t.Fatalf("Could not create factory %v", err)
	}

	c, err := factory.Create()

	if err != nil {
		t.Fatalf("Could not create component %v", err)
	}

	// do a dirty cast for the sake of facilitating the test
	kc := reflect.ValueOf(c).Elem().Interface().(consumer)

	// claim and process the messages and update the counters accordingly
	for _, km := range data.msgs {
		msg, err := claimMessage(ctx, &kc, km)

		if err != nil {
			counter.claimErr++
			println(fmt.Sprintf("Could not claim message %v : %v", msg, err))
			continue
		}

		err = process(&counter, &data)(msg)
		if err != nil {
			println(fmt.Sprintf("Could not process message %v : %v", msg, err))
		}
	}

	assert.Equal(t, counter, data.counter)

}

// some naive decoder implementations for testing

func erroringDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	return func(data []byte, v interface{}) error {
		return errors.New("Predefined Decoder Error")
	}, nil
}

func voidDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	return func(data []byte, v interface{}) error {
		return nil
	}, nil
}

func stringToSliceDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	return func(data []byte, v interface{}) error {
		if arr, ok := v.(*[]string); ok {
			for _, j := range strings.Split(string(data), " ") {
				*arr = append(*arr, j)
			}
		} else {
			return errors.New(fmt.Sprintf("Provided object is not valid for splitting data into a slice '%v'", v))
		}
		return nil
	}, nil
}

func jsonDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	switch contentType {
	case "json":
		return go_json.Unmarshal, nil
	default:
		return nil, fmt.Errorf("Cannot use json decoder for '%s'", contentType)
	}
}

func combinedDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	switch contentType {
	case "json":
		return go_json.Unmarshal, nil
	case "string":
		return stringToSliceDecoder(contentType)
	case "error":
		return erroringDecoder(contentType)
	default:
		return voidDecoder(contentType)
	}
}

var process = func(counter *Counter, data *testingData) func(message async.Message) error {
	return func(message async.Message) error {
		// we always assume we will decode to a slice of strings
		values := []string{}
		// we assume based on our transform function, that we will be able to decode as a rule
		if err := message.Decode(&values); err != nil {
			counter.decodingErr++
			return fmt.Errorf("Error encountered while decoding message from source [%v] : %w", message, err)
		}
		if !reflect.DeepEqual(values, data.dmsgs[counter.messageCount]) {
			counter.resultErr++
			return errors.New(fmt.Sprintf("Could not verify equality for '%v' and '%v' at index '%d'", values, data.dmsgs[counter.messageCount], counter.messageCount))
		}
		counter.messageCount++
		return nil
	}
}
