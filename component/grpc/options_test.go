package grpc

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func TestGRPCOptions(t *testing.T) {
	type args struct {
		options []grpc.ServerOption
	}
	tests := []struct {
		name          string
		args          args
		expectedError error
	}{
		{
			name:          "option used with empty arguments",
			args:          args{},
			expectedError: errors.New("no grpc options provided"),
		},
		{
			name: "option used with non empty arguments",
			args: args{
				[]grpc.ServerOption{grpc.ConnectionTimeout(1 * time.Second)},
			},
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := new(Component)
			err := ServerOptions(tt.args.options...)(comp)
			if tt.expectedError == nil {
				assert.Equal(t, tt.args.options, comp.serverOptions)
			} else {
				assert.Equal(t, err.Error(), tt.expectedError.Error())
			}
		})
	}
}

func TestReflection(t *testing.T) {
	tests := []struct {
		name          string
		expectedError error
	}{
		{
			name:          "option used",
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := new(Component)
			err := Reflection()(comp)
			if tt.expectedError == nil {
				assert.Equal(t, true, comp.enableReflection)
			} else {
				assert.Equal(t, err.Error(), tt.expectedError.Error())
			}
		})
	}
}
