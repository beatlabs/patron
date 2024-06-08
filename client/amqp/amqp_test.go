package amqp

import (
	"context"
	"os"
	"testing"

	metric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var mp *sdkmetric.MeterProvider

func init() {
	var err error
	mp, err = metric.SetupWithMeterProvider(context.Background(), "test", nil)
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	defer mp.Shutdown(context.Background())

	os.Exit(m.Run())
}

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
	exp := tracetest.NewInMemoryExporter()
	_ = patrontrace.Setup("test", nil, exp)

	msg := amqp.Publishing{}
	ctx, sp := injectTraceHeaders(context.Background(), "123", &msg)
	assert.NotNil(t, ctx)
	assert.NotNil(t, sp)
	assert.Len(t, msg.Headers, 2)
	assert.Len(t, exp.GetSpans(), 0)
}
