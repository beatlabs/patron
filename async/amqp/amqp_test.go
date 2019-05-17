package amqp

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func Test_message(t *testing.T) {
	b, err := json.Encode("test")
	assert.NoError(t, err)
	del := &amqp.Delivery{
		Body: b,
	}
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	sp := opentracing.StartSpan("test")
	ctx := context.Background()
	m := message{
		ctx:  ctx,
		del:  del,
		dec:  json.DecodeRaw,
		span: sp,
	}
	assert.Equal(t, ctx, m.Context())
	var data string
	assert.NoError(t, m.Decode(&data))
	assert.Equal(t, "test", data)
	assert.Error(t, m.Ack())
	assert.Error(t, m.Nack())
}

func TestNew(t *testing.T) {

	type args struct {
		url      string
		queue    string
		exchange string
		opt      OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: "e", opt: Buffer(100)}, false},
		{"fail, invalid url", args{url: "", queue: "q", exchange: "e", opt: Buffer(100)}, true},
		{"fail, invalid queue name", args{url: "url", queue: "", exchange: "e", opt: Buffer(100)}, true},
		{"fail, invalid exchange name", args{url: "url", queue: "queue", exchange: "", opt: Buffer(100)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.url, tt.args.queue, tt.args.exchange, tt.args.opt)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestFactory_Create(t *testing.T) {
	type fields struct {
		oo []OptionFunc
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "success", wantErr: false},
		{name: "invalid option", fields: fields{oo: []OptionFunc{Buffer(-10)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{
				url:      "url",
				queue:    "queue",
				exchange: "exchange",
				oo:       tt.fields.oo,
			}
			got, err := f.Create()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_mapHeader(t *testing.T) {
	hh := amqp.Table{"test1": 10, "test2": 0.11}
	mm := map[string]string{"test1": "10", "test2": "0.11"}
	assert.Equal(t, mm, mapHeader(hh))
}

func TestConsumer_Info(t *testing.T) {
	f, err := New("url", "queue", "exchange")
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	expected := make(map[string]interface{})
	expected["type"] = "amqp-consumer"
	expected["queue"] = "queue"
	expected["exchange"] = "exchange"
	expected["requeue"] = true
	expected["buffer"] = 1000
	expected["url"] = "url"
	assert.Equal(t, expected, c.Info())
}
