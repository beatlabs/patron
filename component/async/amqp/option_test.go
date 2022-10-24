package amqp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	type args struct {
		buf int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{buf: 100}, wantErr: false},
		{name: "invalid buffer", args: args{buf: -100}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := consumer{}
			err := WithBuffer(tt.args.buf)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequeue(t *testing.T) {
	c := consumer{}
	err := WithRequeue(false)(&c)
	assert.NoError(t, err)
}

func TestTimeout(t *testing.T) {
	c := consumer{}
	err := WithTimeout(time.Second)(&c)
	assert.NoError(t, err)
}

func TestBindings(t *testing.T) {
	type args struct {
		bindings []string
	}
	tests := []struct {
		name     string
		args     args
		expected []string
		wantErr  bool
	}{
		{name: "multiple bindings", args: args{bindings: []string{"abc", "def"}}, expected: []string{"abc", "def"}, wantErr: false},
		{name: "no bindings", args: args{bindings: []string{}}, expected: []string{""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := consumer{}
			err := WithBindings(tt.args.bindings...)(&c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, c.bindings)
			}
		})
	}
}
