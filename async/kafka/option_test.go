package kafka

import (
	"reflect"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/stretchr/testify/assert"
)

type consumer struct {
	Consumer
	consumerCnf ConsumerConfig
}

func (c *consumer) ConsumerConfig() *ConsumerConfig { return &c.consumerCnf }

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
			c := consumer{}
			err := Buffer(tt.args.buf)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.buf, c.ConsumerConfig().Buffer)
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	c := consumer{}
	c.ConsumerConfig().SaramaConfig = sarama.NewConfig()
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
			c := consumer{}
			c.ConsumerConfig().SaramaConfig = sarama.NewConfig()
			err := Version(tt.args.version)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, c.ConsumerConfig().SaramaConfig.Version)
			}
		})
	}
}

func TestStart(t *testing.T) {
	c := consumer{}
	c.ConsumerConfig().SaramaConfig = sarama.NewConfig()
	err := Start(sarama.OffsetOldest)(&c)
	assert.NoError(t, err)
	assert.Equal(t, sarama.OffsetOldest, c.ConsumerConfig().SaramaConfig.Consumer.Offsets.Initial)
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
			c := consumer{}
			err := Decoder(tt.dec)(&c)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c.ConsumerConfig().DecoderFunc)
				assert.Equal(t,
					reflect.ValueOf(tt.dec).Pointer(),
					reflect.ValueOf(c.ConsumerConfig().DecoderFunc).Pointer(),
				)
			}
		})
	}
}
