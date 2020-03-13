package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractFields(t *testing.T) {
	r, err := http.NewRequest("GET", "/test?value1=1&value2=2", nil)
	assert.NoError(t, err)
	f := extractFields(r)
	assert.Len(t, f, 2)
	assert.Equal(t, "1", f["value1"])
	assert.Equal(t, "2", f["value2"])
}

func Test_extractHeaders(t *testing.T) {
	r, err := http.NewRequest("GET", "/test", nil)
	r.Header.Set("X-HEADER-1", "all capsssss")
	r.Header.Set("X-HEADER-1", "all caps")
	r.Header.Set("x-header-2", "all lower")
	r.Header.Set("X-hEadEr-3", "all mixed")
	r.Header.Set("X-ACME", "")
	assert.NoError(t, err)
	h := extractHeaders(r)
	assert.Len(t, h, 3)
	assert.Equal(t, "all caps", h["X-HEADER-1"])
	assert.Equal(t, "all lower", h["X-HEADER-2"])
	assert.Equal(t, "all mixed", h["X-HEADER-3"])
}

func Test_determineEncoding(t *testing.T) {
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		args    args
		decode  encoding.DecodeFunc
		encode  encoding.EncodeFunc
		ct      string
		wantErr bool
	}{
		{"success json", args{req: request(t, json.Type, json.TypeCharset)}, json.Decode, json.Encode, json.TypeCharset, false},
		{"success json, missing accept", args{req: request(t, json.Type, "")}, json.Decode, json.Encode, json.TypeCharset, false},
		{"success json, missing content type", args{req: request(t, "", json.Type)}, json.Decode, json.Encode, json.TypeCharset, false},
		{"success protobuf", args{req: request(t, protobuf.Type, protobuf.TypeGoogle)}, protobuf.Decode, protobuf.Encode, protobuf.Type, false},
		{"success protobuf, missing accept", args{req: request(t, protobuf.Type, "")}, protobuf.Decode, protobuf.Encode, protobuf.Type, false},
		{"success protobuf, missing content type", args{req: request(t, "", protobuf.Type)}, protobuf.Decode, protobuf.Encode, protobuf.Type, false},
		{"wrong accept", args{req: request(t, json.Type, "xxx")}, nil, nil, json.TypeCharset, true},
		{"missing content header, defaults json", args{req: request(t, "", json.TypeCharset)}, json.Decode, json.Encode, json.TypeCharset, false},
		{"missing headers, defaults json", args{req: request(t, "", "")}, json.Decode, json.Encode, json.TypeCharset, false},
		{"accept */*, defaults to json", args{req: request(t, json.TypeCharset, "*/*")}, json.Decode, json.Encode, json.TypeCharset, false},
		{"wrong content", args{req: request(t, "application/xml", json.TypeCharset)}, nil, nil, json.TypeCharset, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, got, got1, err := determineEncoding(tt.args.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				assert.Nil(t, got1)
				assert.Empty(t, ct)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.NotNil(t, got1)
				assert.Equal(t, tt.ct, ct)
			}
		})
	}
}

func request(t *testing.T, contentType, accept string) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	if contentType != "" {
		req.Header.Set(encoding.ContentTypeHeader, contentType)
	}
	if accept != "" {
		req.Header.Set(encoding.AcceptHeader, accept)
	}
	return req
}

func Test_getOrSetCorrelationID(t *testing.T) {
	withID := http.Header{correlation.HeaderID: []string{"123"}}
	withoutID := http.Header{correlation.HeaderID: []string{}}
	withEmptyID := http.Header{correlation.HeaderID: []string{""}}
	missingHeader := http.Header{}
	type args struct {
		hdr http.Header
	}
	tests := map[string]struct {
		args args
	}{
		"with id":        {args: args{hdr: withID}},
		"without id":     {args: args{hdr: withoutID}},
		"with empty id":  {args: args{hdr: withEmptyID}},
		"missing header": {args: args{hdr: missingHeader}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, getOrSetCorrelationID(tt.args.hdr))
			assert.NotEmpty(t, tt.args.hdr[correlation.HeaderID][0])
		})
	}
}

func Test_handleSuccess(t *testing.T) {
	get, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.NoError(t, err)
	post, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)
	jsonRsp := sync.NewResponse(struct {
		Name    string
		Address string
	}{"Sotiris", "Athens"})
	jsonEncodeFailRsp := sync.NewResponse(struct {
		Name    chan bool
		Address string
	}{nil, "Athens"})

	type args struct {
		req *http.Request
		rsp *sync.Response
		enc encoding.EncodeFunc
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
		wantErr        bool
	}{
		{"GET No Content success", args{req: get, rsp: nil, enc: nil}, http.StatusNoContent, false},
		{"GET OK success", args{req: get, rsp: jsonRsp, enc: json.Encode}, http.StatusOK, false},
		{"POST Created success", args{req: post, rsp: jsonRsp, enc: json.Encode}, http.StatusCreated, false},
		{"Encode failure", args{req: post, rsp: jsonEncodeFailRsp, enc: json.Encode}, http.StatusCreated, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			rsp := httptest.NewRecorder()

			err := handleSuccess(rsp, tt.args.req, tt.args.rsp, tt.args.enc)
			if tt.wantErr {
				assert.Error(t, err)

			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rsp.Code)
			}
		})
	}
}

func Test_handleError(t *testing.T) {
	type args struct {
		err error
		enc encoding.EncodeFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
	}{
		{"bad request", args{err: NewValidationError(), enc: json.Encode}, http.StatusBadRequest},
		{"unauthorized request", args{err: NewUnauthorizedError(), enc: json.Encode}, http.StatusUnauthorized},
		{"forbidden request", args{err: NewForbiddenError(), enc: json.Encode}, http.StatusForbidden},
		{"not found error", args{err: NewNotFoundError(), enc: json.Encode}, http.StatusNotFound},
		{"service unavailable error", args{err: NewServiceUnavailableError(), enc: json.Encode}, http.StatusServiceUnavailable},
		{"internal server error", args{err: NewError(), enc: json.Encode}, http.StatusInternalServerError},
		{"default error", args{err: errors.New("Test"), enc: json.Encode}, http.StatusInternalServerError},
		{"payload encoding error", args{err: NewErrorWithCodeAndPayload(http.StatusBadRequest, make(chan int)), enc: json.Encode}, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp := httptest.NewRecorder()
			handleError(log.Sub(nil), rsp, tt.args.enc, tt.args.err)
			assert.Equal(t, tt.expectedCode, rsp.Code)
		})
	}
}

type testHandler struct {
	err  bool
	resp interface{}
}

func (th testHandler) Process(ctx context.Context, req *sync.Request) (*sync.Response, error) {
	if th.err {
		return nil, errors.New("TEST")
	}
	return sync.NewResponse(th.resp), nil
}

func Test_handler(t *testing.T) {
	require := require.New(t)
	errReq, err := http.NewRequest(http.MethodGet, "/", nil)
	errReq.Header.Set(encoding.ContentTypeHeader, "xml")
	require.NoError(err)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(err)

	req.Header.Set(encoding.ContentTypeHeader, json.Type)
	req.Header.Set(encoding.AcceptHeader, json.Type)

	// success handling
	// failure handling
	type args struct {
		req *http.Request
		hnd sync.ProcessorFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
	}{
		{
			name:         "unsupported content type",
			args:         args{req: errReq, hnd: nil},
			expectedCode: http.StatusUnsupportedMediaType,
		},
		{
			name:         "success handling",
			args:         args{req: req, hnd: testHandler{err: false, resp: "test"}.Process},
			expectedCode: http.StatusOK,
		},
		{
			name:         "error handling",
			args:         args{req: req, hnd: testHandler{err: true, resp: "test"}.Process},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "success handling failed due to encoding",
			args:         args{req: req, hnd: testHandler{err: false, resp: make(chan bool)}.Process},
			expectedCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp := httptest.NewRecorder()
			handler(tt.args.hnd).ServeHTTP(rsp, tt.args.req)
			assert.Equal(t, tt.expectedCode, rsp.Code)
		})
	}
}

func Test_prepareResponse(t *testing.T) {
	rsp := httptest.NewRecorder()
	prepareResponse(rsp, json.TypeCharset)
	assert.Equal(t, json.TypeCharset, rsp.Header().Get(encoding.ContentTypeHeader))
}

func Test_extractParams(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/users/1/status", nil)
	assert.NoError(t, err)
	req.Header.Set(encoding.ContentTypeHeader, json.Type)
	req.Header.Set(encoding.AcceptHeader, json.Type)
	var fields map[string]string

	proc := func(_ context.Context, req *sync.Request) (*sync.Response, error) {
		fields = req.Fields
		return nil, nil
	}

	router := httprouter.New()
	route, err := NewRouteBuilder("/users/:id/status", proc).MethodGet().Build()
	assert.NoError(t, err)
	router.HandlerFunc(route.method, route.path, route.handler)
	router.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "1", fields["id"])
}
