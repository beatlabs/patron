package json

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	j, err := Encode("string")
	require.NoError(t, err)
	b := bytes.NewBuffer(j)
	var data string
	err = Decode(b, &data)
	require.NoError(t, err)
	assert.Equal(t, "string", data)
	err = DecodeRaw(j, &data)
	require.NoError(t, err)
	assert.Equal(t, "string", data)
}
