package group

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	kafkacmp "github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	defaultSaramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-consumer", false)
	require.Nil(t, err)

	brokers := []string{"192.168.1.1"}
	type args struct {
		name      string
		brokers   []string
		topics    []string
		group     string
		saramaCfg *sarama.Config
		options   []kafka.OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "fails with missing name",
			args:    args{name: "", brokers: brokers, topics: []string{"topic1"}, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with missing brokers",
			args:    args{name: "test", brokers: []string{}, topics: []string{"topic1"}, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with empty broker",
			args:    args{name: "test", brokers: []string{" "}, topics: []string{"topic1"}, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with missing topics",
			args:    args{name: "test", brokers: brokers, topics: nil, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with one empty topic",
			args:    args{name: "test", brokers: brokers, topics: []string{"topic1", ""}, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with missing group",
			args:    args{name: "test", brokers: brokers, topics: []string{"topic1"}, group: "", saramaCfg: defaultSaramaCfg},
			wantErr: true,
		},
		{
			name:    "fails with nil Sarama config",
			args:    args{name: "test", brokers: brokers, topics: []string{"topic1"}, group: "group1", saramaCfg: nil},
			wantErr: true,
		},
		{
			name:    "success",
			args:    args{name: "test", brokers: brokers, topics: []string{"topic1"}, group: "group1", saramaCfg: defaultSaramaCfg},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.group, tt.args.topics, tt.args.brokers, tt.args.saramaCfg, tt.args.options...)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

func TestFactory_Create(t *testing.T) {
	t.Parallel()

	type fields struct {
		clientName string
		topics     []string
		brokers    []string
		oo         []kafka.OptionFunc
	}
	tests := map[string]struct {
		fields  fields
		wantErr bool
	}{
		"success": {
			fields: fields{
				clientName: "clientA",
				topics:     []string{"topicA"},
				brokers:    []string{"192.168.1.1"},
			},
			wantErr: false,
		},
		"failed with invalid option": {
			fields: fields{
				clientName: "clientB",
				topics:     []string{"topicA"},
				brokers:    []string{"192.168.1.1"},
				oo:         []kafka.OptionFunc{kafka.Buffer(-100)},
			},
			wantErr: true,
		},
	}
	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig(tt.fields.clientName, false)
			require.Nil(t, err)

			f, err := New(tt.fields.clientName, "no-group", tt.fields.topics, tt.fields.brokers, saramaCfg, tt.fields.oo...)
			require.NoError(t, err)

			got, err := f.Create()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, got)
				consumer, ok := got.(*consumer)
				assert.True(t, ok, "consumer is not of type group.consumer")
				assert.Equal(t, tt.fields.brokers, consumer.config.Brokers)
				assert.Equal(t, tt.fields.topics, consumer.topics)
				assert.True(t, strings.HasSuffix(consumer.config.SaramaConfig.ClientID, tt.fields.clientName), "clientID %q does not have suffix %q", consumer.config.SaramaConfig.ClientID, tt.fields.clientName)
				assert.False(t, got.OutOfOrder())
			}
		})
	}
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
func (m *mockConsumerSession) MarkOffset(string, int32, int64, string) {
}
func (m *mockConsumerSession) Commit() {}
func (m *mockConsumerSession) ResetOffset(string, int32, int64, string) {
}
func (m *mockConsumerSession) MarkMessage(*sarama.ConsumerMessage, string) {}
func (m *mockConsumerSession) Context() context.Context                    { return context.Background() }

func TestHandler_ConsumeClaim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msgs    []*sarama.ConsumerMessage
		error   string
		wantErr bool
	}{
		{"success", saramaConsumerMessages(json.Type), "", false},
		{"failure decoding", saramaConsumerMessages("mock"), "failed to determine decoder for mock", true},
		{"failure content", saramaConsumerMessages(""), "failed to determine content type", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chMsg := make(chan async.Message, 1)
			h := handler{messages: chMsg, consumer: &consumer{}}

			err := h.ConsumeClaim(&mockConsumerSession{}, &mockConsumerClaim{tt.msgs})

			if tt.wantErr {
				require.Error(t, err, tt.error)
			} else {
				require.NoError(t, err)
				ch := <-chMsg
				require.NotNil(t, ch)
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

func TestConsumer_ConsumeFailedBroker(t *testing.T) {
	saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-consumer", false)
	require.NoError(t, err)
	f, err := New("name", "group", []string{"topic"}, []string{"1", "2"}, saramaCfg)
	require.NoError(t, err)
	c, err := f.Create()
	require.NoError(t, err)
	chMsg, chErr, err := c.Consume(context.Background())
	require.Nil(t, chMsg)
	require.Nil(t, chErr)
	require.Error(t, err)
}
