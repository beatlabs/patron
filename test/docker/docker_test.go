package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRuntime(t *testing.T) {
	type args struct {
		expiration time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":          {args: args{expiration: 10 * time.Second}},
		"wrong expiration": {args: args{expiration: -1 * time.Second}, expectedErr: "expiration value is negative"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewRuntime(tt.args.expiration)
			if tt.expectedErr != "" {
				assert.Nil(t, got)
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.expiration, got.expiration)
				assert.NotNil(t, got.pool)
				assert.Equal(t, tt.args.expiration, got.pool.MaxWait)
			}
		})
	}
}
