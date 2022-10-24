package grpc

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestGRPCOptions(t *testing.T) {
	type args struct {
		options []grpc.ServerOption
	}
	tests := map[string]struct {
		args          args
		expectedError error
	}{
		"option used with empty arguments": {args: args{}, expectedError: errors.New("no grpc options provided")},
		"option used with non empty arguments": {args: args{
			[]grpc.ServerOption{grpc.ConnectionTimeout(1 * time.Second)},
		}, expectedError: nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			comp := new(Component)
			err := WithServerOptions(tt.args.options...)(comp)
			if tt.expectedError == nil {
				assert.Equal(t, tt.args.options, comp.serverOptions)
			} else {
				assert.Equal(t, err.Error(), tt.expectedError.Error())
			}
		})
	}
}

func TestReflection(t *testing.T) {
	tests := map[string]struct {
		expectedError error
	}{
		"option used": {expectedError: nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			comp := new(Component)
			err := WithReflection()(comp)
			if tt.expectedError == nil {
				assert.Equal(t, true, comp.enableReflection)
			} else {
				assert.Equal(t, err.Error(), tt.expectedError.Error())
			}
		})
	}
}
