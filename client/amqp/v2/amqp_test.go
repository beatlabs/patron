package v2

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		url string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"fail, missing url": {args: args{}, expectedErr: "url is required"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.args.url)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_injectTraceHeaders(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })
	msg := amqp.Publishing{}
	sp := injectTraceHeaders(context.Background(), "123", &msg)
	assert.NotNil(t, sp)
	assert.NotEmpty(t, msg.Headers)
}
