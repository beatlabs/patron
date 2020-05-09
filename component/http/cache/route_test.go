package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseReadWriter_Header(t *testing.T) {
	rw := newResponseReadWriter()
	rw.Header().Set("key", "value")
	assert.Equal(t, "value", rw.Header().Get("key"))
}

func TestResponseReadWriter_StatusCode(t *testing.T) {
	rw := newResponseReadWriter()
	rw.WriteHeader(100)
	assert.Equal(t, 100, rw.statusCode)
}

func TestResponseReadWriter_ReadWrite(t *testing.T) {
	rw := newResponseReadWriter()
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
	rw := newResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	assert.NoError(t, err)

	b, err := rw.ReadAll()
	assert.NoError(t, err)

	assert.Equal(t, i, len(b))
	assert.Equal(t, str, string(b))
}

func TestResponseReadWriter_ReadAllEmpty(t *testing.T) {
	rw := newResponseReadWriter()

	b, err := rw.ReadAll()
	assert.NoError(t, err)

	assert.Equal(t, 0, len(b))
	assert.Equal(t, "", string(b))
}
