// Package json provides helper functions to handle requests and responses.
package json

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customer struct {
	Name string
}

func TestNewRequest(t *testing.T) {
	t.Parallel()

	got, err := NewRequest(context.Background(), http.MethodPost, "/api/customer", customer{Name: "John Wick"})
	assert.NoError(t, err)
	assert.Equal(t, json.Type, got.Header.Get(encoding.ContentTypeHeader))
	assert.Equal(t, json.Type, got.Header.Get(encoding.AcceptHeader))
	assert.Equal(t, "20", got.Header.Get(encoding.ContentLengthHeader))
}

func TestFromResponse(t *testing.T) {
	t.Parallel()

	expected := customer{Name: "John Wick"}
	buf, err := json.Encode(expected)
	require.NoError(t, err)

	type args struct {
		contentType *string
		payload     []byte
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success ":                            {args: args{contentType: stringPointer(json.Type), payload: buf}},
		"success with charset":                {args: args{contentType: stringPointer(json.TypeCharset), payload: buf}},
		"success with invalid content length": {args: args{contentType: stringPointer(json.Type), payload: buf}},
		"failure, wrong content type":         {args: args{contentType: stringPointer("text/plain"), payload: buf}, expectedErr: "invalid content type provided: text/plain"},
		"failure, empty content type":         {args: args{contentType: stringPointer(""), payload: buf}, expectedErr: "invalid content type provided: "},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rsp := httptest.NewRecorder()
			if tt.args.contentType != nil {
				rsp.Header().Set(encoding.ContentTypeHeader, *tt.args.contentType)
			}

			_, err := rsp.Write(tt.args.payload)
			require.NoError(t, err)

			got := customer{}

			err = FromResponse(context.Background(), rsp.Result(), &got)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			}
		})
	}
}

func stringPointer(val string) *string {
	return &val
}
