package kafka

import (
	"errors"
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
			ab := NewBuilder([]string{}).WithVersion(tt.args.version)
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
			ab := NewBuilder([]string{}).WithTimeout(tt.args.dial)
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
			ab := NewBuilder([]string{}).WithRequiredAcksPolicy(tt.args.requiredAcks)
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
			ab := NewBuilder([]string{}).WithEncoder(tt.args.enc, tt.args.contentType)
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
	seed := createKafkaBroker(t, true)

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
				assert.Empty(t, ab.brokers)
			} else {
				assert.NotEmpty(t, ab.brokers)
			}
		})
	}
}

func Test_createAsyncProducerUsingBuilder(t *testing.T) {
	seed := createKafkaBroker(t, true)

	var builderNoErrors = []error{}
	var builderAllErrors = []error{
		errors.New("brokers list is empty"),
		errors.New("encoder is nil"),
		errors.New("content type is empty"),
	}

	tests := map[string]struct {
		brokers     []string
		version     string
		enc         encoding.EncodeFunc
		contentType string
		wantErrs    []error
	}{
		"success": {
			brokers:     []string{seed.Addr()},
			enc:         json.Encode,
			contentType: json.Type,
			version:     sarama.V0_8_2_0.String(),
			wantErrs:    builderNoErrors,
		},
		"error in all builder steps": {
			brokers:     []string{},
			enc:         nil,
			contentType: "",
			version:     "",
			wantErrs:    builderAllErrors,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotAsyncProducer, gotErrs := NewBuilder(tc.brokers).
				WithVersion(tc.version).
				WithEncoder(tc.enc, tc.contentType).
				Create()

			if len(tc.wantErrs) > 0 {
				assert.ObjectsAreEqual(tc.wantErrs, gotErrs)
				assert.Nil(t, gotAsyncProducer)
			} else {
				assert.NotNil(t, gotAsyncProducer)
				assert.IsType(t, &AsyncProducer{}, gotAsyncProducer)
			}
		})
	}

}
