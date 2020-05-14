package simple

import (
	"testing"

	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	brokers := []string{"192.168.1.1"}
	type args struct {
		name    string
		brokers []string
		topic   string
		options []kafka.OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "fails with missing name",
			args:    args{name: "", brokers: brokers, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with missing brokers",
			args:    args{name: "test", brokers: []string{}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with one empty broker",
			args:    args{name: "test", brokers: []string{""}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with two brokers - one of the is empty",
			args:    args{name: "test", brokers: []string{" ", "broker2"}, topic: "topic1"},
			wantErr: true,
		},
		{
			name:    "fails with missing topics",
			args:    args{name: "test", brokers: brokers, topic: ""},
			wantErr: true,
		},
		{
			name:    "success",
			args:    args{name: "test", brokers: brokers, topic: "topic1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.topic, tt.args.brokers, tt.args.options...)
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
		oo []kafka.OptionFunc
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "success", wantErr: false},
		{name: "failed with invalid option", fields: fields{oo: []kafka.OptionFunc{kafka.Buffer(-100)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{
				name:    "test",
				topic:   "topic",
				brokers: []string{"192.168.1.1"},
				oo:      tt.fields.oo,
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
