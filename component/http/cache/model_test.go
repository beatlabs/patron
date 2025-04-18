package cache

import (
	"testing"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse(t *testing.T) {
	type arg struct {
		payload any
	}

	args := []arg{
		{"string"},
		{10.0},
		{struct {
			a  string
			f  float64
			i  int
			mi map[string]int
			mf map[float64]string
		}{
			a:  "a string",
			f:  12.2,
			i:  22,
			mi: map[string]int{"1": 1},
			mf: map[float64]string{1.1: "1.1"},
		}},
	}

	for _, argument := range args {
		assertForHandlerResponse(t, argument.payload)
	}
}

func assertForHandlerResponse(t *testing.T, payload any) {
	bp, err := json.Encode(payload)
	require.NoError(t, err)

	r := response{
		Response: handlerResponse{
			Bytes:  bp,
			Header: map[string][]string{"header": {"header-value"}},
		},
		LastValid: 10,
		Etag:      "",
		Warning:   "",
		FromCache: false,
		Err:       nil,
	}

	b, err := r.encode()
	require.NoError(t, err)

	rsp := response{}
	err = rsp.decode(b)
	require.NoError(t, err)

	assert.Equal(t, r, rsp)
}
