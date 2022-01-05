package protobuf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/beatlabs/patron/examples"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	user1 := examples.User{
		Firstname: "John",
		Lastname:  "Doe",
	}
	user2 := examples.User{}
	user3 := examples.User{}

	b, err := Encode(&user1)
	assert.NoError(t, err)
	err = DecodeRaw(b, &user2)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetFirstname(), user2.GetFirstname())
	assert.Equal(t, user1.GetLastname(), user2.GetLastname())

	r := bytes.NewReader(b)
	err = Decode(r, &user3)
	assert.NoError(t, err)
	assert.Equal(t, user1.GetFirstname(), user3.GetFirstname())
	assert.Equal(t, user1.GetLastname(), user3.GetLastname())
}

func TestDecodeError(t *testing.T) {
	user := examples.User{}
	err := Decode(errReader(0), &user)
	assert.Error(t, err)
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}
