package protobuf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/beatlabs/patron/examples"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	user1 := examples.User{
		Firstname: proto.String("John"),
		Lastname:  proto.String("Doe"),
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

func TestProtobuf(t *testing.T) {
	user1 := examples.User{
		Firstname: proto.String("John"),
		Lastname:  proto.String("Doe"),
	}

	user2 := examples.User{}

	assert.Equal(t, "", user2.GetFirstname())
	assert.Equal(t, "", user2.GetLastname())

	user1.XXX_DiscardUnknown()
	user1.XXX_Merge(&user2)
	assert.Equal(t, `Firstname:"John" Lastname:"Doe" `, user1.String())
	b, c := user1.Descriptor()
	assert.NotEmpty(t, b)
	assert.Len(t, c, 1)
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}
