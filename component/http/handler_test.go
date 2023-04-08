package http

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
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
	require.NoError(t, err)
	r.Header.Set("X-HEADER-1", "all capsssss")
	r.Header.Set("X-HEADER-1", "all caps")
	r.Header.Set("x-Header-2", "all lower")
	r.Header.Set("X-hEadEr-3", "all mixed")
	r.Header.Set("X-ACME", "")
	assert.NoError(t, err)
	h := extractHeaders(r.Header)
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
		{"missing content Header, defaults json", args{req: request(t, "", json.TypeCharset)}, json.Decode, json.Encode, json.TypeCharset, false},
		{"missing headers, defaults json", args{req: request(t, "", "")}, json.Decode, json.Encode, json.TypeCharset, false},
		{"accept */*, defaults to json", args{req: request(t, json.TypeCharset, "*/*")}, json.Decode, json.Encode, json.TypeCharset, false},
		{"wrong content", args{req: request(t, "application/xml", json.TypeCharset)}, nil, nil, json.TypeCharset, true},
		{"multi-value accept", args{req: request(t, json.TypeCharset, "application/json, */*")}, json.Decode, json.Encode, json.TypeCharset, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ct, got, got1, err := determineEncoding(tt.args.req.Header)
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

func Test_getMultiValueHeaders(t *testing.T) {
	tests := []struct {
		name            string
		headers         string
		expectedHeaders []string
	}{
		{"empty string", "", []string{""}},
		{"single header", "*/*", []string{"*/*"}},
		{"comma separated multi(2) header with space", "application/json, */*", []string{"application/json", "*/*"}},
		{"comma separated multi(2) header with multiple spaces", " application/json, */* ", []string{"application/json", "*/*"}},
		{"comma separated multi(2) header with no space", "application/json,*/*", []string{"application/json", "*/*"}},
		{"comma separated multi(2) header", "application/json,*/*", []string{"application/json", "*/*"}},
		{"comma separated multi(3) header", "application/json,*/*,application/xml", []string{"application/json", "*/*", "application/xml"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			newHeaders := getMultiValueHeaders(tt.headers)
			assert.Equal(t, tt.expectedHeaders, newHeaders)
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

func Test_handleSuccess(t *testing.T) {
	t.Parallel()
	get, err := http.NewRequest(http.MethodGet, "/", nil)
	assert.NoError(t, err)
	post, err := http.NewRequest(http.MethodPost, "/", nil)
	assert.NoError(t, err)
	jsonRsp := NewResponse(struct {
		Name    string
		Address string
	}{"Sotiris", "Athens"})
	jsonEncodeFailRsp := NewResponse(struct {
		Name    chan bool
		Address string
	}{nil, "Athens"})

	type args struct {
		req *http.Request
		rsp *Response
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		name            string
		args            args
		expectedCode    int
		expectedHeaders map[string]string
	}{
		{name: "bad request", args: args{err: NewValidationError(), enc: json.Encode}, expectedCode: http.StatusBadRequest},
		{name: "too many requests with header", args: args{err: NewErrorWithCodeAndPayload(http.StatusTooManyRequests, "test").WithHeaders(map[string]string{"Retry-After": "1628027625"}), enc: json.Encode}, expectedCode: http.StatusTooManyRequests, expectedHeaders: map[string]string{"Retry-After": "1628027625"}},
		{name: "unauthorized request", args: args{err: NewUnauthorizedError(), enc: json.Encode}, expectedCode: http.StatusUnauthorized},
		{name: "forbidden request", args: args{err: NewForbiddenError(), enc: json.Encode}, expectedCode: http.StatusForbidden},
		{name: "not found error", args: args{err: NewNotFoundError(), enc: json.Encode}, expectedCode: http.StatusNotFound},
		{name: "service unavailable error", args: args{err: NewServiceUnavailableError(), enc: json.Encode}, expectedCode: http.StatusServiceUnavailable},
		{name: "internal server error", args: args{err: NewError(), enc: json.Encode}, expectedCode: http.StatusInternalServerError},
		{name: "default error", args: args{err: errors.New("test"), enc: json.Encode}, expectedCode: http.StatusInternalServerError},
		{name: "Payload encoding error", args: args{err: NewErrorWithCodeAndPayload(http.StatusBadRequest, make(chan int)), enc: json.Encode}, expectedCode: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rsp := httptest.NewRecorder()
			handleError(slog.With(), rsp, tt.args.enc, tt.args.err)
			assert.Equal(t, tt.expectedCode, rsp.Code)
			for k, v := range tt.expectedHeaders {
				assert.Equal(t, v, rsp.Header().Get(k))
			}
		})
	}
}

func Test_getOrSetCorrelationID(t *testing.T) {
	t.Parallel()
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
		"missing Header": {args: args{hdr: missingHeader}},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, getOrSetCorrelationID(tt.args.hdr))
			assert.NotEmpty(t, tt.args.hdr[correlation.HeaderID][0])
		})
	}
}

type testHandler struct {
	err  bool
	resp interface{}
}

func (th testHandler) Process(_ context.Context, _ *Request) (*Response, error) {
	if th.err {
		return nil, errors.New("TEST")
	}
	return NewResponse(th.resp), nil
}

func Test_handler(t *testing.T) {
	errReq, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	errReq.Header.Set(encoding.ContentTypeHeader, "xml")
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set(encoding.ContentTypeHeader, json.Type)
	req.Header.Set(encoding.AcceptHeader, json.Type)

	// success handling
	// failure handling
	type args struct {
		req *http.Request
		hnd ProcessorFunc
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
		tt := tt
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

	proc := func(_ context.Context, req *Request) (*Response, error) {
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

func Test_fileserverHandler(t *testing.T) {
	router := httprouter.New()
	path := "/frontend/*path"
	route, err := NewFileServer(path, "testdata", "testdata/index.html").Build()
	require.NoError(t, err)
	router.HandlerFunc(route.method, route.path, route.handler)

	assert.Equal(t, path, route.Path())
	assert.Equal(t, http.MethodGet, route.Method())
	tests := map[string]struct {
		expectedResponse string
		path             string
	}{
		"success":  {path: "/frontend/existing.html", expectedResponse: "existing"},
		"fallback": {path: "/frontend/missing-file", expectedResponse: "fallback"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			// the only way to test do we get the same handler that we provided initially, is to run it explicitly,
			// since all we have in Route itself is a wrapper function
			req, err := http.NewRequest(http.MethodGet, tt.path, nil)
			require.NoError(t, err)

			wr := httptest.NewRecorder()
			router.ServeHTTP(wr, req)
			br, err := ioutil.ReadAll(wr.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedResponse, string(br))
		})
	}
}

func Test_extractParamsRawRoute(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/users/42/status/online", nil)
	assert.NoError(t, err)
	r.Header.Set(encoding.ContentTypeHeader, json.Type)
	r.Header.Set(encoding.AcceptHeader, json.Type)
	var fields map[string]string

	proc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields = ExtractParams(r)
	})

	router := httprouter.New()
	route, err := NewRawRouteBuilder("/users/:id/status/:status", proc).MethodGet().Build()
	assert.NoError(t, err)
	router.HandlerFunc(route.method, route.path, route.handler)
	router.ServeHTTP(httptest.NewRecorder(), r)

	assert.Equal(t, "42", fields["id"])
	assert.Equal(t, "online", fields["status"])
}

func Test_getSingleHeaderEncoding(t *testing.T) {
	testcases := []struct {
		header string
		ct     string
		dec    encoding.DecodeFunc
		enc    encoding.EncodeFunc
		err    error
	}{
		{
			header: json.Type,
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			header: json.TypeCharset,
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			header: "*/*", // json as default (?)
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			header: "*",
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Encoding#:~:text=This%20value%20is%20always%20considered%20as%20acceptable%2C%20even%20if%20omitted
			header: "identity",
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			header: "*/*;q=0.8",
			ct:     json.TypeCharset,
			dec:    json.Decode,
			enc:    json.Encode,
			err:    nil,
		},
		{
			header: protobuf.Type,
			ct:     protobuf.Type,
			dec:    protobuf.Decode,
			enc:    protobuf.Encode,
			err:    nil,
		},
		{
			header: protobuf.TypeGoogle,
			ct:     protobuf.Type,
			dec:    protobuf.Decode,
			enc:    protobuf.Encode,
			err:    nil,
		},
		{
			header: "text/html",
			ct:     "",
			dec:    nil,
			enc:    nil,
			err:    errors.New("accept header not supported"),
		},
		{
			header: "garbage",
			ct:     "",
			dec:    nil,
			enc:    nil,
			err:    errors.New("accept header not supported"),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.header, func(t *testing.T) {
			ct, dec, enc, err := getSingleHeaderEncoding(tc.header)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.ct, ct)

			if reflect.ValueOf(tc.dec).Pointer() != reflect.ValueOf(dec).Pointer() {
				t.Fatalf("Invalid decoder\n\texpected: %v\n\treceived: %v", tc.dec, dec)
			}

			if reflect.ValueOf(tc.enc).Pointer() != reflect.ValueOf(enc).Pointer() {
				t.Fatalf("Invalid encoder\n\texpected: %v\n\treceived: %v", tc.dec, dec)
			}
		})
	}
}
