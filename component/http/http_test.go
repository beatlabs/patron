package http

import (
	"bytes"
	"testing"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	req := NewRequest(nil, nil, nil, nil)
	assert.NotNil(t, req)
}

func TestRequest_Decode(t *testing.T) {
	j, err := json.Encode("string")
	assert.NoError(t, err)
	b := bytes.NewBuffer(j)
	req := NewRequest(nil, b, nil, json.Decode)
	assert.NotNil(t, req)
	var data string
	err = req.Decode(&data)
	assert.NoError(t, err)
	assert.Equal(t, "string", data)
}

func TestNewResponse(t *testing.T) {
	rsp := NewResponse("test")
	assert.NotNil(t, rsp)
	assert.IsType(t, "test", rsp.Payload)
}

func TestResponseReadWriter_Header(t *testing.T) {
	rw := NewResponseReadWriter()
	rw.Header().Set("key", "value")
	assert.Equal(t, "value", rw.Header().Get("key"))
}

func TestResponseReadWriter_StatusCode(t *testing.T) {
	rw := NewResponseReadWriter()
	rw.WriteHeader(100)
	assert.Equal(t, 100, rw.statusCode)
}

func TestResponseReadWriter_ReadWrite(t *testing.T) {
	rw := NewResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	assert.NoError(t, err)

	r := make([]byte, i)
	j, err := rw.Read(r)
	assert.NoError(t, err)

	assert.Equal(t, i, j)
	assert.Equal(t, str, string(r))
}

func TestResponseReadWriter_ReadWriteAll(t *testing.T) {
	rw := NewResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	assert.NoError(t, err)

	b, err := rw.ReadAll()
	assert.NoError(t, err)

	assert.Equal(t, i, len(b))
	assert.Equal(t, str, string(b))
}

func TestResponseReadWriter_ReadAllEmpty(t *testing.T) {
	rw := NewResponseReadWriter()

	b, err := rw.ReadAll()
	assert.NoError(t, err)

	assert.Equal(t, 0, len(b))
	assert.Equal(t, "", string(b))
}
