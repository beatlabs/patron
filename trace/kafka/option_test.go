package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{version: sarama.V0_10_2_0.String()}, wantErr: false},
		{name: "failure, missing version", args: args{version: ""}, wantErr: true},
		{name: "failure, invalid version", args: args{version: "xxxxx"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := sarama.NewConfig()
			ap := &AsyncProducer{cfg: cfg}
			err := Version(tt.args.version)(ap)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				v, err := sarama.ParseKafkaVersion(tt.args.version)
				assert.NoError(t, err)
				assert.Equal(t, v, ap.cfg.Version)
			}
		})
	}
}

func TestTimeouts(t *testing.T) {
	type args struct {
		dial time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{dial: time.Second}, wantErr: false},
		{name: "fail, zero timeout", args: args{dial: 0 * time.Second}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := sarama.NewConfig()
			ap := &AsyncProducer{cfg: cfg}
			err := Timeouts(tt.args.dial)(ap)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.dial, ap.cfg.Net.DialTimeout)
			}
		})
	}
}

func TestRequiredAcksPolicy(t *testing.T) {
	type args struct {
		requiredAcks RequiredAcks
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{requiredAcks: NoResponse}, wantErr: false},
		{name: "success", args: args{requiredAcks: WaitForAll}, wantErr: false},
		{name: "success", args: args{requiredAcks: WaitForLocal}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := AsyncProducer{cfg: sarama.NewConfig()}
			err := RequiredAcksPolicy(tt.args.requiredAcks)(&ap)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncoder(t *testing.T) {
	type args struct {
		enc         encoding.EncodeFunc
		contentType string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "json EncodeFunc", args: args{enc: json.Encode, contentType: json.Type}, wantErr: false},
		{name: "protobuf EncodeFunc", args: args{enc: protobuf.Encode, contentType: protobuf.Type}, wantErr: false},
		{name: "empty content type", args: args{enc: protobuf.Encode, contentType: ""}, wantErr: true},
		{name: "nil EncodeFunc", args: args{enc: nil}, wantErr: true},
		{name: "nil EncodeFunc w/ ct", args: args{enc: nil, contentType: protobuf.Type}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := sarama.NewConfig()
			ap := &AsyncProducer{cfg: cfg}
			err := Encoder(tt.args.enc, tt.args.contentType)(ap)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ap.enc)
				assert.Equal(t, tt.args.contentType, ap.contentType)
			}
		})
	}
}
