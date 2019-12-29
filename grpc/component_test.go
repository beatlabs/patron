package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestCreate(t *testing.T) {
	type args struct {
		port string
	}
	tests := map[string]struct {
		args   args
		expErr string
	}{
		"success":      {args: args{port: ":60000"}},
		"invalid port": {args: args{port: ""}, expErr: "port is empty\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.port).WithOptions(grpc.ConnectionTimeout(1 * time.Second)).Create()
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.port, got.port)
				assert.NotNil(t, got.Server())
			}
		})
	}
}
