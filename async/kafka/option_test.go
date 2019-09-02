package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
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
			c := consumer{}
			err := Buffer(tt.args.buf)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	c := consumer{cfg: sarama.NewConfig()}
	err := Timeout(time.Second)(&c)
	assert.NoError(t, err)
}

func TestVersion(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{version: "1.0.0"}, wantErr: false},
		{name: "failed due to empty", args: args{version: ""}, wantErr: true},
		{name: "failed due to invalid", args: args{version: "1.0.0.0"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New("test", "", "topic", "group", []string{"test"})
			assert.NoError(t, err)
			c, err := f.Create()
			assert.NoError(t, err)
			err = Version(tt.args.version)(c.(*consumer))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStart(t *testing.T) {
	c := consumer{cfg: sarama.NewConfig()}
	err := Start(sarama.OffsetOldest)(&c)
	assert.NoError(t, err)
}
