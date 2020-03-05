package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/trace/kafka"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestNewMessage(t *testing.T) {
	m := kafka.NewMessage("TOPIC", []byte("TEST"))
	assert.Equal(t, "TOPIC", m.Topic)
	assert.Equal(t, []byte("TEST"), m.Body)
}

func TestNewSyncProducer_Failure(t *testing.T) {
	sb := SyncBuilder{kafka.NewBuilder([]string{})}
	got, err := sb.Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	sb := SyncBuilder{kafka.NewBuilder([]string{"xxx"}).WithVersion("xxxx")}
	got, err := sb.Create()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestSyncProducer_SendMessage_Success(t *testing.T) {
	mp := mocks.NewSyncProducer(t, sarama.NewConfig())
	assert.NotNil(t, mp)
	mp.ExpectSendMessageAndSucceed()

	sp := SyncProducer{cfg: sarama.NewConfig(), prod: mp, enc: json.Encode}

	msg := kafka.NewMessage("TOPIC", "TEST")

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

	sp := SyncProducer{cfg: sarama.NewConfig(), tag: opentracing.Tag{Key: "type", Value: "sync"}, prod: mp, enc: json.Encode}

	msg := kafka.NewMessage("TOPIC", "TEST")

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

	sp := SyncProducer{cfg: sarama.NewConfig(), prod: mp, enc: json.Encode}

	msg := kafka.NewMessage("TOPIC", "TEST")

	_, _, err := sp.Send(context.Background(), msg)
	assert.NoError(t, err)

	finishedSpans := mtc.FinishedSpans()
	assert.Equal(t, 1, len(finishedSpans))
	assert.Equal(t, false, finishedSpans[0].Tag("error"))
}
