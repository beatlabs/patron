package kafka

import (
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
