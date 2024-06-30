package protobuf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/beatlabs/patron/encoding/protobuf/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	user1 := test.User{
		Firstname: "John",
		Lastname:  "Doe",
	}
	user2 := test.User{}
	user3 := test.User{}

	b, err := Encode(&user1)
	require.NoError(t, err)
	err = DecodeRaw(b, &user2)
	require.NoError(t, err)
	assert.Equal(t, user1.GetFirstname(), user2.GetFirstname())
	assert.Equal(t, user1.GetLastname(), user2.GetLastname())

	r := bytes.NewReader(b)
	err = Decode(r, &user3)
	require.NoError(t, err)
	assert.Equal(t, user1.GetFirstname(), user3.GetFirstname())
	assert.Equal(t, user1.GetLastname(), user3.GetLastname())
}

func TestDecodeError(t *testing.T) {
	user := test.User{}
	err := Decode(errReader(0), &user)
	require.Error(t, err)
}

type errReader int

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("test error")
}
