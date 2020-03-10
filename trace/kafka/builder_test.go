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
	seed := createKafkaBroker(t)
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
			ab := NewBuilder([]string{seed.Addr()}).WithVersion(tt.args.version)
			if tt.wantErr {
				assert.NotEmpty(t, ab.Errors)
			} else {
				assert.Empty(t, ab.Errors)
				v, err := sarama.ParseKafkaVersion(tt.args.version)
				assert.NoError(t, err)
				assert.Equal(t, v, ab.Cfg.Version)
			}
		})
	}
}

func TestTimeouts(t *testing.T) {
	seed := createKafkaBroker(t)
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
			ab := NewBuilder([]string{seed.Addr()}).WithTimeout(tt.args.dial)
			if tt.wantErr {
				assert.NotEmpty(t, ab.Errors)
			} else {
				assert.Empty(t, ab.Errors)
				assert.Equal(t, tt.args.dial, ab.Cfg.Net.DialTimeout)
			}
		})
	}
}

func createKafkaBroker(t *testing.T) *sarama.MockBroker {
	lead := sarama.NewMockBroker(t, 2)
	metadataResponse := new(sarama.MetadataResponse)
	metadataResponse.AddBroker(lead.Addr(), lead.BrokerID())
	metadataResponse.AddTopicPartition("TOPIC", 0, lead.BrokerID(), nil, nil, sarama.ErrNoError)

	prodSuccess := new(sarama.ProduceResponse)
	prodSuccess.AddTopicPartition("TOPIC", 0, sarama.ErrDuplicateSequenceNumber)

	lead.Returns(prodSuccess)

	config := sarama.NewConfig()
	config.Producer.Flush.Messages = 10
	config.Producer.Return.Successes = true
	seed := sarama.NewMockBroker(t, 1)
	seed.Returns(metadataResponse)
	return seed
}

func TestRequiredAcksPolicy(t *testing.T) {
	seed := createKafkaBroker(t)
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
			ab := NewBuilder([]string{seed.Addr()}).WithRequiredAcksPolicy(tt.args.requiredAcks)
			if tt.wantErr {
				assert.NotEmpty(t, ab.Errors)
			} else {
				assert.Empty(t, ab.Errors)
				assert.EqualValues(t, tt.args.requiredAcks, ab.Cfg.Producer.RequiredAcks)
			}
		})
	}
}

func TestEncoder(t *testing.T) {
	seed := createKafkaBroker(t)
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
			ab := NewBuilder([]string{seed.Addr()}).WithEncoder(tt.args.enc, tt.args.contentType)
			if tt.wantErr {
				assert.NotEmpty(t, ab.Errors)
			} else {
				assert.Empty(t, ab.Errors)
				assert.NotNil(t, ab.Enc)
				assert.Equal(t, tt.args.contentType, ab.ContentType)
			}
		})
	}
}

func TestBrokers(t *testing.T) {
	seed := createKafkaBroker(t)

	type args struct {
		brokers []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "single mock broker", args: args{brokers: []string{seed.Addr()}}, wantErr: false},
		{name: "multiple mock brokers", args: args{brokers: []string{seed.Addr(), seed.Addr(), seed.Addr()}}, wantErr: false},
		{name: "empty brokers list", args: args{brokers: []string{}}, wantErr: true},
		{name: "nil brokers list", args: args{brokers: nil}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ab := NewBuilder(tt.args.brokers)
			if tt.wantErr {
				assert.Empty(t, ab.Brokers)
			} else {
				assert.NotEmpty(t, ab.Brokers)
			}
		})
	}
}
