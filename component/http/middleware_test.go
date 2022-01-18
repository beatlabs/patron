package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A middleware generator that tags resp for assertions
func tagMiddleware(tag string) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(tag))
			// next
			h.ServeHTTP(w, r)
		})
	}
}

// Panic middleware to test recovery middleware
func panicMiddleware(v interface{}) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(v)
		})
	}
}

func getMockLimiter(allow bool) *rate.Limiter {
	if allow {
		return rate.NewLimiter(1, 1)
	}
	return rate.NewLimiter(1, 0)
}

func TestMiddlewareChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"middleware 1,2,3 and finish", args{next: handler, mws: []MiddlewareFunc{t1, t2, t3}}, 202, "t1\nt2\nt3\n"},
		{"middleware 1,2 and finish", args{next: handler, mws: []MiddlewareFunc{t1, t2}}, 202, "t1\nt2\n"},
		{"no middleware and finish", args{next: handler, mws: []MiddlewareFunc{}}, 202, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

func TestMiddlewares(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"auth middleware success", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{success: true})}}, 202, ""},
		{"auth middleware false", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{success: false})}}, 401, "Unauthorized\n"},
		{"auth middleware error", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{err: errors.New("auth error")})}}, 500, "Internal Server Error\n"},
		{"tracing middleware", args{next: handler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index", statusCodeLoggerHandler{})}}, 202, ""},
		{"rate limiting middleware", args{next: handler, mws: []MiddlewareFunc{NewRateLimitingMiddleware(getMockLimiter(true))}}, 202, ""},
		{"rate limiting middleware error", args{next: handler, mws: []MiddlewareFunc{NewRateLimitingMiddleware(getMockLimiter(false))}}, 429, "Requests greater than limit\n"},
		{"recovery middleware from panic 1", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware("error")}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 2", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(errors.New("error"))}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 3", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(-1)}}, 500, "Internal Server Error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

// TestSpanLogError tests whether an HTTP handler with a tracing middleware adds a log event in case of we return an error.
func TestSpanLogError(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name                 string
		args                 args
		expectedCode         int
		expectedBody         string
		expectedSpanLogError string
	}{
		{"tracing middleware - error", args{next: errorHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index", statusCodeLoggerHandler{})}}, http.StatusInternalServerError, "foo", "foo"},
		{"tracing middleware - success", args{next: successHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index", statusCodeLoggerHandler{})}}, http.StatusOK, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mtr.Reset()
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())

			if tt.expectedSpanLogError != "" {
				require.Equal(t, 1, len(mtr.FinishedSpans()))
				spanLogError := getSpanLogError(t, mtr.FinishedSpans()[0])
				assert.Equal(t, tt.expectedSpanLogError, spanLogError)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rc := httptest.NewRecorder()
	rw := newResponseWriter(rc, true)

	_, err := rw.Write([]byte("test"))
	assert.NoError(t, err)
	rw.WriteHeader(202)

	assert.Equal(t, 202, rw.status, "status expected 202 but got %d", rw.status)
	assert.Len(t, rw.Header(), 1, "Header count expected to be 1")
	assert.True(t, rw.statusHeaderWritten, "expected to be true")
	assert.Equal(t, "test", rc.Body.String(), "body expected to be test but was %s", rc.Body.String())
}

func TestStripQueryString(t *testing.T) {
	t.Parallel()
	type args struct {
		path string
	}
	tests := map[string]struct {
		args         args
		expectedPath string
		expectedErr  error
	}{
		"query string 1": {
			args: args{
				path: "foo?bar=value1&baz=value2",
			},
			expectedPath: "foo",
		},
		"query string 2": {
			args: args{
				path: "/foo?bar=value1&baz=value2",
			},
			expectedPath: "/foo",
		},
		"query string 3": {
			args: args{
				path: "http://foo/bar?baz=value1",
			},
			expectedPath: "http://foo/bar",
		},
		"no query string": {
			args: args{
				path: "http://foo/bar",
			},
			expectedPath: "http://foo/bar",
		},
		"empty": {
			args: args{
				path: "",
			},
			expectedPath: "",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s, err := stripQueryString(tt.args.path)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPath, s)
			}
		})
	}
}

func getSpanLogError(t *testing.T, span *mocktracer.MockSpan) string {
	logs := span.Logs()
	if len(logs) == 0 {
		assert.FailNow(t, "empty logs")
		return ""
	}

	for _, log := range logs {
		for _, field := range log.Fields {
			if field.Key == fieldNameError {
				return field.ValueString
			}
		}
	}

	assert.FailNowf(t, "missing logs", "missing field %s", fieldNameError)
	return ""
}

func TestNewCompressionMiddleware(t *testing.T) {
	tests := map[string]struct {
		cm MiddlewareFunc
	}{
		"gzip":    {cm: NewCompressionMiddleware(8)},
		"deflate": {cm: NewCompressionMiddleware(8)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Content-Length", "123")
				w.WriteHeader(202)
			})
			req, err := http.NewRequest("GET", "/test", nil)
			assert.NoError(t, err)

			req.Header.Set("Accept-Encoding", name)
			compressionMiddleware := tc.cm
			assert.NoError(t, err)
			assert.NotNil(t, compressionMiddleware)

			rc := httptest.NewRecorder()
			compressionMiddleware(handler).ServeHTTP(rc, req)
			actual := rc.Header().Get("Content-Encoding")
			assert.Equal(t, name, actual)

			cl := rc.Header().Get("Content-Length")
			assert.Equal(t, "", cl)
		})
	}
}

func TestNewCompressionMiddlewareServer(t *testing.T) {
	tests := []struct {
		cm               MiddlewareFunc
		status           int
		acceptEncoding   string
		expectedEncoding string
	}{
		{
			status:           200,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           201,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           204,
			acceptEncoding:   "gzip",
			expectedEncoding: "",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           304,
			acceptEncoding:   "gzip",
			expectedEncoding: "",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           404,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           200,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           201,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           204,
			acceptEncoding:   "deflate",
			expectedEncoding: "",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           304,
			acceptEncoding:   "deflate",
			expectedEncoding: "",
			cm:               NewCompressionMiddleware(8),
		},
		{
			status:           404,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               NewCompressionMiddleware(8),
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d - %s", tc.status, tc.expectedEncoding), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})

			compressionMiddleware := tc.cm
			assert.NotNil(t, compressionMiddleware)
			s := httptest.NewServer(compressionMiddleware(handler))
			defer s.Close()

			req, err := http.NewRequest("GET", s.URL, nil)
			assert.NoError(t, err)
			req.Header.Set("Accept-Encoding", tc.acceptEncoding)

			resp, err := s.Client().Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedEncoding, resp.Header.Get("Content-Encoding"))
		})
	}
}

func TestNewCompressionMiddleware_Ignore(t *testing.T) {
	var ceh string // accept-encoding, content type

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
	middleware := NewCompressionMiddleware(8, "/metrics")

	assert.NotNil(t, middleware)

	// check if the route actually ignored
	req1, err := http.NewRequest("GET", "/metrics", nil)
	assert.NoError(t, err)
	req1.Header.Set("Accept-Encoding", "gzip")
	assert.NoError(t, err)

	rc1 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc1, req1)

	ceh = rc1.Header().Get("Content-Encoding")
	assert.NotNil(t, ceh)
	assert.Equal(t, ceh, "")

	// check if other routes remains untouched
	req2, err := http.NewRequest("GET", "/alive", nil)
	assert.NoError(t, err)
	req2.Header.Set("Accept-Encoding", "gzip")

	rc2 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc2, req2)

	ceh = rc2.Header().Get("Content-Encoding")
	assert.NotNil(t, ceh)
	assert.Equal(t, "gzip", ceh)
}

func TestNewCompressionMiddleware_Headers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := NewCompressionMiddleware(8, "/metrics")

	tests := map[string]struct {
		cm               MiddlewareFunc
		statusCode       int
		encodingExpected string
	}{
		"gzip":                {cm: middleware, statusCode: http.StatusOK, encodingExpected: gzipHeader},
		"deflate":             {cm: middleware, statusCode: http.StatusOK, encodingExpected: deflateHeader},
		"gzip, *":             {cm: middleware, statusCode: http.StatusOK, encodingExpected: gzipHeader},
		"deflate, *":          {cm: middleware, statusCode: http.StatusOK, encodingExpected: deflateHeader},
		"invalid, gzip, *":    {cm: middleware, statusCode: http.StatusOK, encodingExpected: gzipHeader},
		"invalid, deflate, *": {cm: middleware, statusCode: http.StatusOK, encodingExpected: deflateHeader},
		"invalid":             {cm: middleware, statusCode: http.StatusNotAcceptable, encodingExpected: ""},
		"invalid, *":          {cm: middleware, statusCode: http.StatusOK, encodingExpected: ""},
		"identity":            {cm: middleware, statusCode: http.StatusOK, encodingExpected: identityHeader},
		"gzip, identity":      {cm: middleware, statusCode: http.StatusOK, encodingExpected: gzipHeader},
		"*":                   {cm: middleware, statusCode: http.StatusOK, encodingExpected: ""},
		"":                    {cm: middleware, statusCode: http.StatusOK, encodingExpected: identityHeader},
		"not present":         {cm: middleware, statusCode: http.StatusOK, encodingExpected: identityHeader},
	}

	for encodingName, tc := range tests {
		t.Run(fmt.Sprintf("%q: compression middleware acts according the Accept-Encoding header", encodingName), func(t *testing.T) {
			require.NotNil(t, tc.cm)
			// given
			req1, err := http.NewRequest("GET", "/alive", nil)
			require.NoError(t, err)
			if encodingName != "not present" {
				req1.Header.Set("Accept-Encoding", encodingName)
			}

			// when
			rc1 := httptest.NewRecorder()
			tc.cm(handler).ServeHTTP(rc1, req1)

			// then
			assert.Equal(t, tc.statusCode, rc1.Code)

			contentEncodingHeader := rc1.Header().Get("Content-Encoding")
			assert.NotNil(t, contentEncodingHeader)
			assert.Equal(t, tc.encodingExpected, contentEncodingHeader)
		})
	}
}

func TestSelectEncoding(t *testing.T) {
	tests := []struct {
		optionalName string
		given        string
		expected     string
		isErr        bool
	}{
		{given: "", expected: "identity", optionalName: "is empty but present, only identity"},

		{given: "*", expected: "*"},
		{given: "gzip", expected: "gzip"},
		{given: "deflate", expected: "deflate"},

		{given: "whatever", expected: "", isErr: true, optionalName: "whatever, not supported"},
		{given: "whatever, *", expected: "*", optionalName: "whatever, but also a star"},

		{given: "gzip, deflate", expected: "gzip"},
		{given: "whatever, gzip, deflate", expected: "gzip"},
		{given: "gzip, whatever, deflate", expected: "gzip"},
		{given: "gzip, deflate, whatever", expected: "gzip"},

		{given: "gzip,deflate", expected: "gzip"},
		{given: "gzip,whatever,deflate", expected: "gzip"},
		{given: "whatever,gzip,deflate", expected: "gzip"},
		{given: "gzip,deflate,whatever", expected: "gzip"},

		{given: "deflate, gzip", expected: "deflate"},
		{given: "whatever, deflate, gzip", expected: "deflate"},
		{given: "deflate, whatever, gzip", expected: "deflate"},
		{given: "deflate, gzip, whatever", expected: "deflate"},

		{given: "deflate, gzip", expected: "deflate"},
		{given: "whatever,deflate,gzip", expected: "deflate"},
		{given: "deflate,whatever,gzip", expected: "deflate"},
		{given: "deflate,gzip,whatever", expected: "deflate"},

		{given: "gzip;q=1.0, deflate;q=1.0", expected: "gzip", optionalName: "equal weights"},
		{given: "deflate;q=1.0, gzip;q=1.0", expected: "deflate", optionalName: "equal weights 2"},

		{given: "gzip;q=1.0, deflate;q=0.5", expected: "gzip"},
		{given: "gzip;q=1.0, deflate;q=0.5, *;q=0.2", expected: "gzip"},
		{given: "deflate;q=1.0, gzip;q=0.5", expected: "deflate"},
		{given: "deflate;q=1.0, gzip;q=0.5, *;q=0.2", expected: "deflate"},

		{given: "gzip;q=0.5, deflate;q=1.0", expected: "deflate"},
		{given: "gzip;q=0.5, deflate;q=1.0, *;q=0.2", expected: "deflate"},
		{given: "deflate;q=0.5, gzip;q=1.0", expected: "gzip"},
		{given: "deflate;q=0.5, gzip;q=1.0, *;q=0.2", expected: "gzip"},

		{given: "whatever;q=1.0, *;q=0.2", expected: "*"},

		{given: "deflate, gzip;q=1.0", expected: "deflate"},
		{given: "deflate, gzip;q=0.5", expected: "deflate"},

		{given: "deflate;q=0.5, gzip", expected: "gzip"},

		{given: "deflate;q=0.5, gzip;q=-0.5", expected: "deflate"},
		{given: "deflate;q=0.5, gzip;q=1.5", expected: "gzip"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("encoding %q is parsed as %s ; error is expected: %t ; %s", tc.given, tc.expected, tc.isErr, tc.optionalName), func(t *testing.T) {
			// when
			result, err := parseAcceptEncoding(tc.given)

			// then
			assert.Equal(t, tc.isErr, err != nil)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSupported(t *testing.T) {
	tests := []struct {
		algorithm   string
		isSupported bool
	}{
		{algorithm: "gzip", isSupported: true},
		{algorithm: "deflate", isSupported: true},
		{algorithm: "*", isSupported: true},
		{algorithm: "something else", isSupported: false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q check results in %t", tc.algorithm, tc.isSupported), func(t *testing.T) {
			// when
			result := !notSupportedCompression(tc.algorithm)

			// then
			assert.Equal(t, result, tc.isSupported)
		})
	}
}

func TestParseWeights(t *testing.T) {
	tests := []struct {
		priorityStr string
		expected    float64
	}{
		{priorityStr: "q=1.0", expected: 1.0},
		{priorityStr: "q=0.5", expected: 0.5},
		{priorityStr: "q=-0.5", expected: 0.0},
		{priorityStr: "q=1.5", expected: 1.0},
		{priorityStr: "q=", expected: 1.0},
		{priorityStr: "", expected: 1.0},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("for given priority: %q, expect %f", tc.priorityStr, tc.expected), func(t *testing.T) {
			// when
			result := parseWeight(tc.priorityStr)

			// then
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSelectByWeight(t *testing.T) {
	tests := []struct {
		name     string
		given    map[float64]string
		expected string
		isErr    bool
	}{
		{
			name:     "sorted map",
			given:    map[float64]string{1.0: "gzip", 0.5: "deflate"},
			expected: "gzip",
		},
		{
			name:     "not sorted map",
			given:    map[float64]string{0.5: "gzip", 1.0: "deflate"},
			expected: "deflate",
		},
		{
			name:     "empty weights map",
			given:    map[float64]string{},
			expected: "",
			isErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			selected, err := selectByWeight(tc.given)

			// then
			assert.Equal(t, tc.isErr, err != nil)
			assert.Equal(t, tc.expected, selected)
		})
	}
}

func TestAddWithWeight(t *testing.T) {
	tests := []struct {
		name        string
		weightedMap map[float64]string
		weight      float64
		algorithm   string
		expected    map[float64]string
	}{
		{
			name:        "empty",
			weightedMap: map[float64]string{},
			weight:      1.0,
			algorithm:   "gzip",
			expected:    map[float64]string{1.0: "gzip"},
		},
		{
			name:        "new",
			weightedMap: map[float64]string{1.0: "gzip"},
			weight:      0.5,
			algorithm:   "deflate",
			expected:    map[float64]string{1.0: "gzip", 0.5: "deflate"},
		},
		{
			name:        "already exists",
			weightedMap: map[float64]string{1.0: "gzip"},
			weight:      1.0,
			algorithm:   "deflate",
			expected:    map[float64]string{1.0: "gzip"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			addWithWeight(tc.weightedMap, tc.weight, tc.algorithm)

			// then
			assert.Equal(t, tc.expected, tc.weightedMap)
		})
	}
}

func TestIsConnectionReset(t *testing.T) {
	tests := map[string]struct {
		err      error
		expected bool
	}{
		"Broken pipe": {
			err:      errors.New("blah blah broken pipe blah blah"),
			expected: true,
		},
		"connection reset": {
			err:      errors.New("blah blah connection reset blah blah"),
			expected: true,
		},
		"read: connection reset": {
			err:      errors.New("blah blah read: connection reset blah blah"),
			expected: false,
		},
		"b00m random error": {
			err:      errors.New("blah blah b00m random error blah blah"),
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			result := isErrConnectionReset(tc.err)

			// then
			assert.Equal(t, tc.expected, result)
		})
	}
}

type failWriter struct{}

func (fw *failWriter) Header() http.Header {
	return http.Header{}
}

func (fw *failWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("foo")
}

func (fw *failWriter) WriteHeader(statusCode int) {
}

func TestSetResponseWriterStatusOnResponseFailWrite(t *testing.T) {
	failWriter := &failWriter{}
	failDynamicCompressionResponseWriter := &dynamicCompressionResponseWriter{failWriter, "", nil, 0, deflateLevel}

	tests := []struct {
		Name           string
		ResponseWriter *responseWriter
	}{
		{
			Name:           "Failing responseWriter with http.ResponseWriter",
			ResponseWriter: newResponseWriter(failWriter, false),
		},
		{
			Name:           "Failing responseWriter with http.ResponseWriter",
			ResponseWriter: newResponseWriter(failDynamicCompressionResponseWriter, false),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			_, err := test.ResponseWriter.Write([]byte(`"foo":"bar"`))
			assert.Error(t, err)
			assert.Equal(t, http.StatusOK, test.ResponseWriter.status)
		})
	}
}
