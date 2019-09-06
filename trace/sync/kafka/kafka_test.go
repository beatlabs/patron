package kafka

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/beatlabs/patron/errors"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMessage(t *testing.T) {
	m := NewMessage("TOPIC", []byte("TEST"))
	assert.Equal(t, "TOPIC", m.topic)
	assert.Equal(t, []byte("TEST"), m.body)
}

func TestNewJSONMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{name: "failure due to invalid data", data: make(chan bool), wantErr: true},
		{name: "success", data: "TEST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewJSONMessage("TOPIC", tt.data)
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
	got, err := NewSyncProducer([]string{})
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	got, err := NewSyncProducer([]string{"xxx"}, Version("xxxx"))
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSyncProducer_SendMessage_Success(t *testing.T) {
	mp := mocks.NewSyncProducer(t, sarama.NewConfig())
	assert.NotNil(t, mp)
	mp.ExpectSendMessageAndSucceed()

	sp := SyncProducer{cfg: sarama.NewConfig(), prod: mp}

	msg, err := NewJSONMessage("TOPIC", "TEST")
	assert.NoError(t, err)

	partition, offset, err := sp.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NotEqual(t, int32(-1), partition)
	assert.NotEqual(t, int64(-1), offset)
	assert.NoError(t, sp.Close())
}

func TestSyncProducer_SendMessage_Failure(t *testing.T) {
	mp := mocks.NewSyncProducer(t, sarama.NewConfig())
	assert.NotNil(t, mp)
	mp.ExpectSendMessageAndFail(errors.New("some error"))

	sp := SyncProducer{cfg: sarama.NewConfig(), tag: opentracing.Tag{Key: "type", Value: "sync"}, prod: mp}

	msg, err := NewJSONMessage("TOPIC", "TEST")
	assert.NoError(t, err)

	partition, offset, err := sp.Send(context.Background(), msg)
	assert.Error(t, err)
	assert.Equal(t, int32(-1), partition)
	assert.Equal(t, int64(-1), offset)
	assert.NoError(t, sp.Close())
}

func TestSyncProducer_SendMessage_Tracing(t *testing.T) {
	mp := mocks.NewSyncProducer(t, sarama.NewConfig())
	assert.NotNil(t, mp)
	mp.ExpectSendMessageAndSucceed()

	mtc := mocktracer.New()
	opentracing.SetGlobalTracer(mtc)

	sp := SyncProducer{cfg: sarama.NewConfig(), prod: mp}

	msg, err := NewJSONMessage("TOPIC", "TEST")
	assert.NoError(t, err)

	_, _, err = sp.Send(context.Background(), msg)
	assert.NoError(t, err)

	finishedSpans := mtc.FinishedSpans()
	assert.Equal(t, 1, len(finishedSpans))
	assert.Equal(t, false, finishedSpans[0].Tag("error"))
}
