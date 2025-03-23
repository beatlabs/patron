package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	target = "target"
)

func TestNewClient(t *testing.T) {
	type args struct {
		opts []grpc.DialOption
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				opts: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
			},
		},
		"failure missing grpc.WithInsecure()": {
			args:        args{},
			expectedErr: "grpc: no transport security set (use grpc.WithTransportCredentials(insecure.NewCredentials()) explicitly or set credentials)",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotConn, err := NewClient(target, tt.args.opts...)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, gotConn)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gotConn)
			}
		})
	}
}
