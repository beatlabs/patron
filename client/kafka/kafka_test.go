package kafka

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewMessage(t *testing.T) {
	type args struct {
		topic string
		body  interface{}
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":       {args: args{topic: "TOPIC", body: []byte("BODY")}},
		"invalid topic": {args: args{topic: "", body: []byte("BODY")}, expectedErr: "topic is empty"},
		"invalid body":  {args: args{topic: "TOPIC", body: nil}, expectedErr: "body is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewMessage(tt.args.topic, tt.args.body)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.topic, got.topic)
				assert.Equal(t, tt.args.body, got.body)
				assert.Nil(t, got.key)
				assert.Nil(t, got.cloudEvt)
			}
		})
	}
}

func TestNewMessageWithKey(t *testing.T) {
	tests := map[string]struct {
		topic       string
		body        interface{}
		key         string
		expectedErr string
	}{
		"success":       {topic: "TOPIC", body: []byte("TEST"), key: "TEST"},
		"invalid topic": {topic: "", body: []byte("BODY"), key: "TEST", expectedErr: "topic is empty"},
		"invalid body":  {topic: "TOPIC", body: nil, key: "TEST", expectedErr: "body is nil"},
		"invalid key":   {topic: "TOPIC", body: []byte("TEST"), key: "", expectedErr: "key is empty"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewMessageWithKey(tt.topic, tt.body, tt.key)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.topic, got.topic)
				assert.Equal(t, tt.body, got.body)
				assert.Equal(t, tt.key, *got.key)
				assert.Nil(t, got.cloudEvt)
			}
		})
	}
}

func TestMessageFromCloudEvent(t *testing.T) {
	evt := cloudevents.NewEvent()
	type args struct {
		topic string
		evt   *cloudevents.Event
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":       {args: args{topic: "TOPIC", evt: &evt}},
		"invalid topic": {args: args{topic: "", evt: &evt}, expectedErr: "topic is empty"},
		"invalid event": {args: args{topic: "TOPIC", evt: nil}, expectedErr: "cloud event is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := MessageFromCloudEvent(tt.args.topic, tt.args.evt)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.topic, got.topic)
				assert.Equal(t, tt.args.evt, got.cloudEvt)
				assert.Nil(t, got.body)
				assert.Nil(t, got.key)
			}
		})
	}
}

func TestNewAsyncProducer_Failure(t *testing.T) {
	got, chErr, err := NewBuilder([]string{}).CreateAsync()
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Nil(t, chErr)
}

func TestNewAsyncProducer_Option_Failure(t *testing.T) {
	got, chErr, err := NewBuilder([]string{"xxx"}).WithVersion("xxxx").CreateAsync()
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Nil(t, chErr)
}

func TestNewSyncProducer_Failure(t *testing.T) {
	got, err := NewBuilder([]string{}).CreateSync()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	got, err := NewBuilder([]string{"xxx"}).WithVersion("xxxx").CreateSync()
	assert.Error(t, err)
	assert.Nil(t, got)
}
