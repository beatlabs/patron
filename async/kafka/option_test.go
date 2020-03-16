package kafka

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	type args struct {
		buf int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{buf: 100}, wantErr: false},
		{name: "invalid buffer", args: args{buf: -100}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ConsumerConfig{}
			err := Buffer(tt.args.buf)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.buf, c.Buffer)
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	c := ConsumerConfig{}
	c.SaramaConfig = sarama.NewConfig()
	err := Timeout(time.Second)(&c)
	assert.NoError(t, err)
}

func TestVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		expected sarama.KafkaVersion
	}{
		{name: "success", args: args{version: "2.1.0"}, wantErr: false, expected: sarama.V2_1_0_0},
		{name: "failed due to empty", args: args{version: ""}, wantErr: true},
		{name: "failed due to invalid", args: args{version: "1.0.0.0"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ConsumerConfig{}
			c.SaramaConfig = sarama.NewConfig()
			err := Version(tt.args.version)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, c.SaramaConfig.Version)
			}
		})
	}
}

func TestStart(t *testing.T) {
	tests := map[string]struct {
		optionFunc      OptionFunc
		expectedOffsets int64
	}{
		"Start": {
			Start(5),
			int64(5),
		},
		"StartFromNewest": {
			StartFromNewest(),
			sarama.OffsetNewest,
		},
		"StartFromOldest": {
			StartFromOldest(),
			sarama.OffsetOldest,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := ConsumerConfig{}
			c.SaramaConfig = sarama.NewConfig()
			err := tt.optionFunc(&c)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOffsets, c.SaramaConfig.Consumer.Offsets.Initial)
		})
	}
}

func TestDecoder1(t *testing.T) {

	tests := []struct {
		name string
		dec  encoding.DecodeRawFunc
		err  bool
	}{
		{
			name: "test simple decoder",
			dec: func(data []byte, v interface{}) error {
				return nil
			},
			err: false,
		},
		{
			name: "test nil decoder",
			dec:  nil,
			err:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ConsumerConfig{}
			err := Decoder(tt.dec)(&c)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c.DecoderFunc)
				assert.Equal(t,
					reflect.ValueOf(tt.dec).Pointer(),
					reflect.ValueOf(c.DecoderFunc).Pointer(),
				)
			}
		})
	}
}

func TestDecoderJSON(t *testing.T) {
	c := ConsumerConfig{}
	err := DecoderJSON()(&c)
	assert.NoError(t, err)
	assert.Equal(t,
		reflect.ValueOf(json.DecodeRaw).Pointer(),
		reflect.ValueOf(c.DecoderFunc).Pointer(),
	)
}

func TestTopics(t *testing.T) {
	tcases := []struct {
		name    string
		topics  []string
		wantErr error
	}{
		{
			name:    "all topics are empty",
			topics:  []string{"", ""},
			wantErr: errors.New("one of the topics values is empty"),
		},
		{
			name:    "one of the topics is empty",
			topics:  []string{"", "value"},
			wantErr: errors.New("one of the topics values is empty"),
		},
		{
			name:    "one of the topics is only-spaces value",
			topics:  []string{"     ", "value"},
			wantErr: errors.New("one of the topics values is empty"),
		},
		{
			name:   "all topics are non-empty",
			topics: []string{"value1", "value2"},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			c := ConsumerConfig{}
			err := Topics(tc.topics)(&c)
			if tc.wantErr != nil {
				assert.Error(t, tc.wantErr, err.Error())
				assert.Empty(t, c.Topics)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.topics, c.Topics)
			}
		})
	}
}

func TestBrokers(t *testing.T) {
	tcases := []struct {
		name    string
		brokers []string
		wantErr error
	}{
		{
			name:    "all brokers are empty",
			brokers: []string{"", ""},
			wantErr: errors.New("one of the brokers values is empty"),
		},
		{
			name:    "one of the brokers is empty",
			brokers: []string{"", "value"},
			wantErr: errors.New("one of the brokers values is empty"),
		},
		{
			name:    "one of the brokers is only-spaces value",
			brokers: []string{"value1", "     ", "value2"},
			wantErr: errors.New("one of the brokers values is empty"),
		},
		{
			name:    "all brokers are non-empty",
			brokers: []string{"value1", "value2"},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			c := ConsumerConfig{}
			err := Brokers(tc.brokers)(&c)
			if tc.wantErr != nil {
				assert.Error(t, tc.wantErr, err.Error())
				assert.Empty(t, c.Brokers)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.brokers, c.Brokers)
			}
		})
	}
}

func Test_containsEmptyValue(t *testing.T) {
	tcases := []struct {
		name       string
		values     []string
		wantResult bool
	}{
		{
			name:       "all values are empty",
			values:     []string{"", ""},
			wantResult: true,
		},
		{
			name:       "one of the values is empty",
			values:     []string{"", "value"},
			wantResult: true,
		},
		{
			name:       "one of the values is only-spaces value",
			values:     []string{"     ", "value"},
			wantResult: true,
		},
		{
			name:       "all values are non-empty",
			values:     []string{"value1", "value2"},
			wantResult: false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantResult, containsEmtpyValue(tc.values))
		})
	}
}
