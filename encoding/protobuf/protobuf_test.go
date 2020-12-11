package protobuf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golang/protobuf/proto"
)

func TestEncodeDecode(t *testing.T) {
	test := Test{
		Label: proto.String("hello"),
		Type:  proto.Int32(17),
		Reps:  []int64{1, 2, 3},
	}
	test2 := Test{}
	test3 := Test{}

	b, err := Encode(&test)
	assert.NoError(t, err)
	err = DecodeRaw(b, &test2)
	assert.NoError(t, err)
	assert.Equal(t, test.GetLabel(), test2.GetLabel())
	assert.Equal(t, test.GetType(), test2.GetType())
	assert.Equal(t, test.GetReps(), test2.GetReps())

	r := bytes.NewReader(b)
	err = Decode(r, &test3)
	assert.NoError(t, err)
	assert.Equal(t, test.GetLabel(), test3.GetLabel())
	assert.Equal(t, test.GetType(), test3.GetType())
	assert.Equal(t, test.GetReps(), test3.GetReps())
}

func TestDecodeError(t *testing.T) {
	test := Test{}
	err := Decode(errReader(0), &test)
	assert.Error(t, err)
}

func TestProtobuf(t *testing.T) {
	test := Test{
		Label: proto.String("hello"),
		Type:  proto.Int32(17),
		Reps:  []int64{1, 2, 3},
	}

	test1 := Test{
		Type: nil,
	}

	assert.Equal(t, "", test1.GetLabel())
	assert.Equal(t, int32(0), test1.GetType())
	assert.Equal(t, []int64(nil), test1.GetReps())

	test.XXX_DiscardUnknown()
	test.XXX_Merge(&test1)
	assert.Equal(t, "label:\"hello\" type:17 reps:1 reps:2 reps:3 ", test.String())
	b, c := test.Descriptor()
	assert.NotEmpty(t, b)
	assert.Len(t, c, 1)
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}
