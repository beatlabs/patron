package http

import (
	"testing"

	"github.com/beatlabs/patron/encoding/json"

	"github.com/stretchr/testify/assert"
)

func TestCachedResponse(t *testing.T) {

	type arg struct {
		payload interface{}
	}

	args := []arg{
		{"string"},
		{10.0},
		//{10},
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
		assertForCacheHandlerResponse(t, argument.payload)
	}

}

func assertForCacheHandlerResponse(t *testing.T, payload interface{}) {

	bp, err := json.Encode(payload)
	assert.NoError(t, err)

	response := CachedResponse{
		Response: CacheHandlerResponse{
			Bytes:  bp,
			Header: map[string]string{"header": "header-value"},
		},
		LastValid: 10,
		Etag:      "",
		Warning:   "",
		FromCache: false,
		Err:       nil,
	}

	b, err := response.Encode()
	assert.NoError(t, err)

	rsp := CachedResponse{}
	err = rsp.Decode(b)
	assert.NoError(t, err)

	assert.Equal(t, response, rsp)

}
