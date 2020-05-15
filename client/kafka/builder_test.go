package kafka

import (
	"testing"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"

	"github.com/Shopify/sarama"
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
			ab := NewBuilder([]string{"123"}).WithVersion(tt.args.version)
			if tt.wantErr {
				assert.NotEmpty(t, ab.errors)
			} else {
				assert.Empty(t, ab.errors)
				v, err := sarama.ParseKafkaVersion(tt.args.version)
				assert.NoError(t, err)
				assert.Equal(t, v, ab.cfg.Version)
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
			ab := NewBuilder([]string{"123"}).WithTimeout(tt.args.dial)
			if tt.wantErr {
				assert.NotEmpty(t, ab.errors)
			} else {
				assert.Empty(t, ab.errors)
				assert.Equal(t, tt.args.dial, ab.cfg.Net.DialTimeout)
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
		{name: "failure", args: args{requiredAcks: -5}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ab := NewBuilder([]string{"123"}).WithRequiredAcksPolicy(tt.args.requiredAcks)
			if tt.wantErr {
				assert.NotEmpty(t, ab.errors)
			} else {
				assert.Empty(t, ab.errors)
				assert.EqualValues(t, tt.args.requiredAcks, ab.cfg.Producer.RequiredAcks)
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
			ab := NewBuilder([]string{"123"}).WithEncoder(tt.args.enc, tt.args.contentType)
			if tt.wantErr {
				assert.NotEmpty(t, ab.errors)
			} else {
				assert.Empty(t, ab.errors)
				assert.NotNil(t, ab.enc)
				assert.Equal(t, tt.args.contentType, ab.contentType)
			}
		})
	}
}

func TestBrokers(t *testing.T) {

	type args struct {
		brokers []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "single mock broker", args: args{brokers: []string{"123"}}, wantErr: false},
		{name: "multiple mock brokers", args: args{brokers: []string{"123", "123", "123"}}, wantErr: false},
		{name: "empty brokers list", args: args{brokers: []string{}}, wantErr: true},
		{name: "brokers list with an empty value", args: args{brokers: []string{" ", "value"}}, wantErr: true},
		{name: "nil brokers list", args: args{brokers: nil}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ab := NewBuilder(tt.args.brokers)
			if tt.wantErr {
				assert.NotEmpty(t, ab.errors)
			} else {
				assert.Empty(t, ab.errors)
			}
		})
	}
}
