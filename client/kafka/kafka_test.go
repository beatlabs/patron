package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMessage(t *testing.T) {
	m := NewMessage("TOPIC", []byte("TEST"))
	assert.Equal(t, "TOPIC", m.topic)
	assert.Equal(t, []byte("TEST"), m.body)
}

func TestNewMessageWithKey(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		key     string
		wantErr bool
	}{
		{name: "success", data: []byte("TEST"), key: "TEST"},
		{name: "failure due to empty message key", data: []byte("TEST"), key: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMessageWithKey("TOPIC", tt.data, tt.key)
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

func TestNewAsyncProducer_Failure(t *testing.T) {
	got, chErr, err := NewBuilder([]string{}).CreateAsync()
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Nil(t, chErr)
}

func TestNewAsyncProducer_Option_Failure(t *testing.T) {
	got, chErr, err := NewBuilder([]string{"xxx"}).WithVersion("xxxx").CreateAsync()
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Nil(t, chErr)
}

func TestNewSyncProducer_Failure(t *testing.T) {
	got, err := NewBuilder([]string{}).CreateSync()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewSyncProducer_Option_Failure(t *testing.T) {
	got, err := NewBuilder([]string{"xxx"}).WithVersion("xxxx").CreateSync()
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNewMessageWithHeader(t *testing.T) {
	tests := []struct {
		name                 string
		data                 []byte
		setHeaderKeys        []string
		setHeaderValues      []string
		expectedHeaderKeys   []string
		expectedHeaderValues []string
	}{
		{
			name: "2-headers", data: []byte("TEST"),
			setHeaderKeys: []string{"header1", "header2"}, setHeaderValues: []string{"value1", "value2"},
			expectedHeaderKeys: []string{"header1", "header2"}, expectedHeaderValues: []string{"value1", "value2"},
		},
		{
			name: "2-headers", data: []byte("TEST"),
			setHeaderKeys: []string{"header1", "header1"}, setHeaderValues: []string{"value1", "value2"},
			expectedHeaderKeys: []string{"header1", "header1"}, expectedHeaderValues: []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage("TOPIC", tt.data)

			// set message headers
			for i := 0; i < len(tt.setHeaderKeys); i++ {
				msg.SetHeader(tt.setHeaderKeys[i], tt.setHeaderValues[i])
			}

			// verify
			for i := 0; i < len(tt.expectedHeaderKeys); i++ {
				assert.Equal(t, string(msg.headers[i].Key), tt.expectedHeaderKeys[i])
				assert.Equal(t, string(msg.headers[i].Value), tt.expectedHeaderValues[i])
			}
		})
	}
}
