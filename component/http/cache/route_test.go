package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	r := make([]byte, i)
	j, err := rw.Read(r)
	require.NoError(t, err)

	assert.Equal(t, i, j)
	assert.Equal(t, str, string(r))
}

func TestResponseReadWriter_ReadWriteAll(t *testing.T) {
	rw := newResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	require.NoError(t, err)

	b, err := rw.ReadAll()
	require.NoError(t, err)

	assert.Len(t, b, i)
	assert.Equal(t, str, string(b))
}

func TestResponseReadWriter_ReadAllEmpty(t *testing.T) {
	rw := newResponseReadWriter()

	b, err := rw.ReadAll()
	require.NoError(t, err)

	assert.Empty(t, b)
	assert.Equal(t, "", string(b))
}
