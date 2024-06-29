// Package observability provides functionality for initializing OpenTelemetry's traces and metrics.
// It includes methods for setting up and shutting down the observability components.
package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestComponentAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("component", "test"), ComponentAttribute("test"))
}

func TestClientAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("client", "test"), ClientAttribute("test"))
}

func TestStatusAttribute(t *testing.T) {
	type args struct {
		err error
	}
	tests := map[string]struct {
		args args
		want attribute.KeyValue
	}{
		"succeeded": {args: args{err: nil}, want: SucceededAttribute},
		"failed":    {args: args{err: assert.AnError}, want: FailedAttribute},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, StatusAttribute(tt.args.err))
		})
	}
}
