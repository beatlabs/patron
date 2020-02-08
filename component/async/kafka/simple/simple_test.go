package simple

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/stretchr/testify/assert"
)

const fooTopic = "foo_topic"

func TestNew(t *testing.T) {
	brokers := []string{"192.168.1.1"}
	type args struct {
		name    string
		brokers []string
		topic   string
		options []kafka.OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "fails with missing name",
			args:    args{name: "", brokers: brokers, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with missing brokers",
			args:    args{name: "test", brokers: []string{}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with one empty broker",
			args:    args{name: "test", brokers: []string{""}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with two brokers - one of the is empty",
			args:    args{name: "test", brokers: []string{" ", "broker2"}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with missing topics",
			args:    args{name: "test", brokers: brokers, topic: ""},
			wantErr: true,
		},
		{
			name:    "success",
			args:    args{name: "test", brokers: brokers, topic: "topic1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.topic, tt.args.brokers, tt.args.options...)
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
		oo []kafka.OptionFunc
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "success", wantErr: false},
		{name: "failed with invalid option", fields: fields{oo: []kafka.OptionFunc{kafka.Buffer(-100)}}, wantErr: true},
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

func newBroker(t *testing.T, topic string) *sarama.MockBroker {
	broker := sarama.NewMockBroker(t, 0)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetVersion(1).
			SetOffset(topic, 0, sarama.OffsetNewest, 10).
			SetOffset(topic, 0, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetVersion(4).
			SetMessage(topic, 0, 10, sarama.StringEncoder(`"Foo"`)),
	})

	return broker
}

func consume(t *testing.T, f *Factory) (context.Context, async.Consumer, <-chan async.Message, <-chan error) {
	c, err := f.Create()
	assert.NoError(t, err)
	ctx := context.Background()
	chMsg, chErr, err := c.Consume(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, chMsg)
	assert.NotNil(t, chErr)

	return ctx, c, chMsg, chErr
}

func TestConsumer_ConsumeFromOldest(t *testing.T) {
	broker := newBroker(t, fooTopic)

	f, err := New("name", fooTopic, []string{broker.Addr()}, kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
	assert.NoError(t, err)

	ctx, c, chMsg, chErr := consume(t, f)

	select {
	case msg := <-chMsg:
		var str string
		err = msg.Decode(&str)
		assert.NoError(t, err)
		assert.Equal(t, "Foo", str)
	case err = <-chErr:
		t.Fatal(err)
	}

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()

	ctx.Done()
}

func TestConsumer_ClaimMessageError(t *testing.T) {
	broker := newBroker(t, fooTopic)

	f, err := New("name", fooTopic, []string{broker.Addr()}, kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
	assert.NoError(t, err)

	ctx, c, chMsg, chErr := consume(t, f)

	select {
	case <-chMsg:
		t.Error("Message arrived in message channel")
	case err = <-chErr:
		assert.Error(t, err)
	}

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()

	ctx.Done()
}

func TestConsumer_ConsumerError(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(fooTopic, 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetVersion(1).
			SetOffset(fooTopic, 0, sarama.OffsetNewest, 10).
			SetOffset(fooTopic, 0, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetVersion(0).
			SetMessage(fooTopic, 0, 10, sarama.StringEncoder(`"Foo"`)),
	})

	f, err := New("name", fooTopic, []string{broker.Addr()}, kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
	assert.NoError(t, err)

	ctx, c, chMsg, chErr := consume(t, f)

	select {
	case <-chMsg:
		t.Error("Message arrived in message channel")
	case err = <-chErr:
		assert.Error(t, err)
	}

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()

	ctx.Done()
}

func TestConsumer_LeaderNotAvailableError(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(fooTopic, 0, 123),
	})

	f, err := New("name", fooTopic, []string{broker.Addr()}, kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
	assert.NoError(t, err)

	c, err := f.Create()
	assert.NoError(t, err)
	ctx := context.Background()

	_, _, err = c.Consume(ctx)
	assert.Error(t, err)

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()
}

func TestConsumer_NoLeaderError(t *testing.T) {
	broker := sarama.NewMockBroker(t, 0)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()),
	})

	f, err := New("name", fooTopic, []string{broker.Addr()}, kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
	assert.NoError(t, err)

	c, err := f.Create()
	assert.NoError(t, err)
	ctx := context.Background()

	_, _, err = c.Consume(ctx)
	assert.Error(t, err)

	err = c.Close()
	assert.NoError(t, err)
	broker.Close()
}
