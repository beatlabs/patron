// Package json contains helper methods to handler requests and responses more easily.
package json

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customer struct {
	Name string
}

func TestReadRequest(t *testing.T) {
	t.Parallel()

	expected := customer{Name: "John Wick"}
	buf, err := json.Encode(expected)
	require.NoError(t, err)

	type args struct {
		header string
		body   []byte
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success, no header":           {args: args{header: "", body: buf}},
		"success, single star header":  {args: args{header: "*", body: buf}},
		"success, double star header":  {args: args{header: "*/*", body: buf}},
		"success, json header":         {args: args{header: json.Type, body: buf}},
		"success, json charset header": {args: args{header: json.TypeCharset, body: buf}},
		"failure, invalid header":      {args: args{header: "text/xml", body: buf}, expectedErr: "invalid content type provided: text/xml"},
		"failure, invalid body":        {args: args{header: "", body: []byte("-")}, expectedErr: "unexpected EOF"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got customer

			req := createRequest(t, tt.args.body, tt.args.header)

			err := ReadRequest(req, &got)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			}
		})
	}
}

func createRequest(t *testing.T, buf []byte, header string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/api/customer", bytes.NewBuffer(buf))
	require.NoError(t, err)

	if header == "" {
		return req
	}

	req.Header.Set(encoding.ContentTypeHeader, header)

	return req
}

func TestWriteResponse(t *testing.T) {
	t.Parallel()

	expected := customer{Name: "John Wick"}
	expectedBuf, err := json.Encode(expected)
	require.NoError(t, err)

	rsp := httptest.NewRecorder()
	err = WriteResponse(rsp, http.StatusOK, expected)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rsp.Code)
	assert.Equal(t, expectedBuf, rsp.Body.Bytes())
	assert.Equal(t, json.Type, rsp.Header().Get(encoding.ContentTypeHeader))
	assert.Equal(t, strconv.FormatInt(int64(len(expectedBuf)), 10), rsp.Header().Get(encoding.ContentLengthHeader))
}

func TestValidateAcceptHeader(t *testing.T) {
	t.Parallel()

	expected := customer{Name: "John Wick"}

	type args struct {
		acceptHeader *string
		status       int
		payload      interface{}
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success, missing header":      {args: args{acceptHeader: nil, status: http.StatusOK, payload: expected}},
		"success, empty header":        {args: args{acceptHeader: stringPointer(""), status: http.StatusOK, payload: expected}},
		"success, single star header":  {args: args{acceptHeader: stringPointer(singleStar), status: http.StatusOK, payload: expected}},
		"success, double star header":  {args: args{acceptHeader: stringPointer(doubleStar), status: http.StatusOK, payload: expected}},
		"success, identity header":     {args: args{acceptHeader: stringPointer("identity"), status: http.StatusOK, payload: expected}},
		"success, json header":         {args: args{acceptHeader: stringPointer(json.Type), status: http.StatusOK, payload: expected}},
		"success, json charset header": {args: args{acceptHeader: stringPointer(json.TypeCharset), status: http.StatusOK, payload: expected}},
		"success, multi header":        {args: args{acceptHeader: stringPointer(fmt.Sprintf("%s,%s", singleStar, json.Type)), status: http.StatusOK, payload: expected}},
		"failure, invalid header":      {args: args{acceptHeader: stringPointer("-"), status: http.StatusOK, payload: expected}, expectedErr: "invalid accept header: -"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(http.MethodPost, "/api/customer", nil)
			require.NoError(t, err)
			if tt.args.acceptHeader != nil {
				req.Header.Set(encoding.AcceptHeader, *tt.args.acceptHeader)
			}

			err = ValidateAcceptHeader(req)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func stringPointer(val string) *string {
	return &val
}
