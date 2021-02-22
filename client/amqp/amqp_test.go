package amqp

import (
	"testing"

	"github.com/beatlabs/patron/examples"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestNewMessage(t *testing.T) {
	m := NewMessage("xxx", []byte("test"))
	assert.Equal(t, "xxx", m.contentType)
	assert.Equal(t, []byte("test"), m.body)
}

func TestNewJSONMessage(t *testing.T) {
	m, err := NewJSONMessage("xxx")
	assert.NoError(t, err)
	assert.Equal(t, "application/json", m.contentType)
	assert.Equal(t, []byte(`"xxx"`), m.body)
	_, err = NewJSONMessage(make(chan bool))
	assert.Error(t, err)
}

func TestNewProtobufMessage(t *testing.T) {
	u := examples.User{
		Firstname: proto.String("John"),
		Lastname:  proto.String("Doe"),
	}
	m, err := NewProtobufMessage(&u)
	assert.NoError(t, err)
	assert.Equal(t, "application/x-protobuf", m.contentType)
	assert.Len(t, m.body, 11)
	u = examples.User{}
	_, err = NewProtobufMessage(&u)
	assert.Error(t, err)
}

func TestNewPublisher(t *testing.T) {
	type args struct {
		url string
		exc string
		opt OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "fail, missing url", args: args{}, wantErr: true},
		{name: "fail, missing exchange", args: args{url: "url"}, wantErr: true},
		{name: "fail, missing exchange", args: args{url: "url", exc: "exc", opt: Timeout(0)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPublisher(tt.args.url, tt.args.exc, tt.args.opt)
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
