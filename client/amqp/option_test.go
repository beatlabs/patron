package amqp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	type args struct {
		timeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{timeout: time.Second}, wantErr: false},
		{name: "failure, invalid timeout", args: args{timeout: 0 * time.Second}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := TracedPublisher{cfg: defaultCfg}
			err := WithTimeout(tt.args.timeout)(&p)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
