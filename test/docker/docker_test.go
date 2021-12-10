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
		tst := tt
		t.Run(name, func(t *testing.T) {
			got, err := NewRuntime(tst.args.expiration)
			if tst.expectedErr != "" {
				assert.Nil(t, got)
				assert.EqualError(t, err, tst.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tst.args.expiration, got.Expiration())
				assert.NotNil(t, got.Pool())
				assert.Empty(t, got.Resources())
				assert.Equal(t, tst.args.expiration, got.pool.MaxWait)
			}
		})
	}
}
