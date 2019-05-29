package amqp

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/thebeatapp/patron/encoding/json"
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
		exchange Exchange
		bindings []string
		opt      OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success, direct exchange", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: Exchange{Name: "e", Kind: amqp.ExchangeDirect}, bindings: []string{}, opt: Buffer(100)}, false},
		{"success, fanout exchange", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: Exchange{Name: "e", Kind: amqp.ExchangeFanout}, bindings: []string{}, opt: Buffer(100)}, false},
		{"success, topic exchange", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: Exchange{Name: "e", Kind: amqp.ExchangeTopic}, bindings: []string{}, opt: Buffer(100)}, false},
		{"success, headers exchange", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: Exchange{Name: "e", Kind: amqp.ExchangeHeaders}, bindings: []string{}, opt: Buffer(100)}, false},
		{"fail, invalid url", args{url: "", queue: "q", exchange: Exchange{Name: "e", Kind: amqp.ExchangeFanout}, bindings: []string{}, opt: Buffer(100)}, true},
		{"fail, invalid queue Name", args{url: "url", queue: "", exchange: Exchange{Name: "e", Kind: amqp.ExchangeFanout}, bindings: []string{}, opt: Buffer(100)}, true},
		{"fail, missing exchange Name", args{url: "url", queue: "queue", exchange: Exchange{"", ""}, bindings: []string{}, opt: Buffer(100)}, true},
		{"fail, missing exchange type", args{url: "url", queue: "queue", exchange: Exchange{"e", ""}, bindings: []string{}, opt: Buffer(100)}, true},
		{"fail, invalid exchange type", args{url: "url", queue: "queue", exchange: Exchange{"e", "foobar"}, bindings: []string{}, opt: Buffer(100)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.url, tt.args.queue, tt.args.exchange, tt.args.bindings, tt.args.opt)
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
				exchange: Exchange{Name: "exchange", Kind: amqp.ExchangeFanout},
				bindings: []string{},
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
	f, err := New("url", "queue", Exchange{"exchange", amqp.ExchangeTopic}, []string{})
	assert.NoError(t, err)
	c, err := f.Create()
	assert.NoError(t, err)
	expected := make(map[string]interface{})
	expected["type"] = "amqp-consumer"
	expected["queue"] = "queue"
	expected["exchange"] = Exchange{"exchange", amqp.ExchangeTopic}
	expected["requeue"] = true
	expected["buffer"] = 1000
	expected["url"] = "url"
	assert.Equal(t, expected, c.Info())
}
