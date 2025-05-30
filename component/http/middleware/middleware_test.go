package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	httpcache "github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"golang.org/x/time/rate"
)

type stubAuthenticator struct {
	success bool
	err     error
}

func (mo stubAuthenticator) Authenticate(_ *http.Request) (bool, error) {
	if mo.err != nil {
		return false, mo.err
	}
	return mo.success, nil
}

// A middleware generator that tags resp for assertions.
func tagMiddleware(tag string) Func {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(tag))
			// next
			h.ServeHTTP(w, r)
		})
	}
}

// Panic middleware to test recovery middleware.
func panicMiddleware(v any) Func {
	return func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	r, err := http.NewRequest(http.MethodPost, "/test", nil)
	require.NoError(t, err)

	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	type args struct {
		next http.Handler
		mws  []Func
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"middleware 1,2,3 and finish", args{next: handler, mws: []Func{t1, t2, t3}}, 202, "t1\nt2\nt3\n"},
		{"middleware 1,2 and finish", args{next: handler, mws: []Func{t1, t2}}, 202, "t1\nt2\n"},
		{"no middleware and finish", args{next: handler, mws: []Func{}}, 202, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = Chain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

func TestMiddlewares(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	r, err := http.NewRequest(http.MethodPost, "/test", nil)
	require.NoError(t, err)

	loggingTracingMiddleware, err := NewLoggingTracing("/index", StatusCodeLoggerHandler{})
	require.NoError(t, err)
	rateLimitingWithLimitMiddleware, err := NewRateLimiting(getMockLimiter(true))
	require.NoError(t, err)
	rateLimitingWithoutLimitMiddlware, err := NewRateLimiting(getMockLimiter(false))
	require.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []Func
	}

	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"auth middleware success", args{next: handler, mws: []Func{NewAuth(&stubAuthenticator{success: true})}}, 202, ""},
		{"auth middleware false", args{next: handler, mws: []Func{NewAuth(&stubAuthenticator{success: false})}}, 401, "Unauthorized\n"},
		{"auth middleware error", args{next: handler, mws: []Func{NewAuth(&stubAuthenticator{err: errors.New("auth error")})}}, 500, "Internal Server Error\n"},
		{"tracing middleware", args{next: handler, mws: []Func{loggingTracingMiddleware}}, 202, ""},
		{"rate limiting middleware", args{next: handler, mws: []Func{rateLimitingWithLimitMiddleware}}, 202, ""},
		{"rate limiting middleware error", args{next: handler, mws: []Func{rateLimitingWithoutLimitMiddlware}}, 429, "Requests greater than limit\n"},
		{"recovery middleware from panic 1", args{next: handler, mws: []Func{NewRecovery(), panicMiddleware("error")}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 2", args{next: handler, mws: []Func{NewRecovery(), panicMiddleware(errors.New("error"))}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 3", args{next: handler, mws: []Func{NewRecovery(), panicMiddleware(-1)}}, 500, "Internal Server Error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = Chain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

func TestNewLoggingTracing(t *testing.T) {
	type args struct {
		path             string
		statusCodeLogger StatusCodeLoggerHandler
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"empty path should return error":          {args: args{path: "", statusCodeLogger: StatusCodeLoggerHandler{}}, expectedErr: "path cannot be empty"},
		"valid path should succeed without error": {args: args{path: "/path", statusCodeLogger: StatusCodeLoggerHandler{}}, expectedErr: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			loggingTracingMiddleware, err := NewLoggingTracing(tt.args.path, tt.args.statusCodeLogger)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, loggingTracingMiddleware)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, loggingTracingMiddleware)
			}
		})
	}
}

// TestSpanLogError tests whether an HTTP handler with a tracing middleware adds a log event in case of we return an error.
func TestSpanLogError(t *testing.T) {
	// Setup tracing
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("foo"))
		assert.NoError(t, err)
	})

	r, err := http.NewRequest(http.MethodPost, "/test", nil)
	require.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []Func
	}
	loggingTracingMiddleware, err := NewLoggingTracing("/index", StatusCodeLoggerHandler{})
	require.NoError(t, err)

	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{
			"tracing middleware - error",
			args{next: errorHandler, mws: []Func{loggingTracingMiddleware}},
			http.StatusInternalServerError, "foo",
		},
		{
			"tracing middleware - success",
			args{next: successHandler, mws: []Func{loggingTracingMiddleware}},
			http.StatusOK, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer exp.Reset()

			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc, true)
			tt.args.next = Chain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			snaps := exp.GetSpans().Snapshots()
			assert.Len(t, snaps, 1)
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rc := httptest.NewRecorder()
	rw := newResponseWriter(rc, true)

	_, err := rw.Write([]byte("test"))
	require.NoError(t, err)
	rw.WriteHeader(http.StatusAccepted)

	assert.Equal(t, http.StatusAccepted, rw.status, "status expected 202 but got %d", rw.status)
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

func TestNewCompressionMiddleware(t *testing.T) {
	type args struct {
		deflateLevel int
	}
	tests := map[string]struct {
		args                  args
		compressionTypeHeader string
		expectedErr           string
	}{
		"less than min level should return error":               {args: args{deflateLevel: -3}, expectedErr: "invalid compression level -3: want value in range [-2, 9]"},
		"greater than max level should return error":            {args: args{deflateLevel: 10}, expectedErr: "invalid compression level 10: want value in range [-2, 9]"},
		"deflate - level in range should succeed without error": {args: args{deflateLevel: 1}, compressionTypeHeader: deflateHeader, expectedErr: ""},
		"gzip - level in range should succeed without error":    {args: args{deflateLevel: 1}, compressionTypeHeader: gzipHeader, expectedErr: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			compressionMiddleware, err := NewCompression(tt.args.deflateLevel)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, compressionMiddleware)
				return
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Add("Content-Length", "123")
				w.WriteHeader(http.StatusAccepted)
			})
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			require.NoError(t, err)

			req.Header.Set("Accept-Encoding", tt.compressionTypeHeader)

			rc := httptest.NewRecorder()
			compressionMiddleware(handler).ServeHTTP(rc, req)
			actual := rc.Header().Get("Content-Encoding")
			assert.Equal(t, tt.compressionTypeHeader, actual)

			cl := rc.Header().Get("Content-Length")
			assert.Empty(t, cl)
		})
	}
}

func TestNewCompressionMiddlewareServer(t *testing.T) {
	compressionMiddleware, err := NewCompression(8)
	require.NoError(t, err)

	tests := []struct {
		cm               Func
		status           int
		acceptEncoding   string
		expectedEncoding string
	}{
		{
			status:           200,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               compressionMiddleware,
		},
		{
			status:           201,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               compressionMiddleware,
		},
		{
			status:           204,
			acceptEncoding:   "gzip",
			expectedEncoding: "",
			cm:               compressionMiddleware,
		},
		{
			status:           304,
			acceptEncoding:   "gzip",
			expectedEncoding: "",
			cm:               compressionMiddleware,
		},
		{
			status:           404,
			acceptEncoding:   "gzip",
			expectedEncoding: "gzip",
			cm:               compressionMiddleware,
		},
		{
			status:           200,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               compressionMiddleware,
		},
		{
			status:           201,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               compressionMiddleware,
		},
		{
			status:           204,
			acceptEncoding:   "deflate",
			expectedEncoding: "",
			cm:               compressionMiddleware,
		},
		{
			status:           304,
			acceptEncoding:   "deflate",
			expectedEncoding: "",
			cm:               compressionMiddleware,
		},
		{
			status:           404,
			acceptEncoding:   "deflate",
			expectedEncoding: "deflate",
			cm:               compressionMiddleware,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d - %s", tt.status, tt.expectedEncoding), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})

			compressionMiddleware := tt.cm
			assert.NotNil(t, compressionMiddleware)
			s := httptest.NewServer(compressionMiddleware(handler))
			defer s.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, s.URL, nil)
			require.NoError(t, err)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)

			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedEncoding, resp.Header.Get("Content-Encoding"))
			require.NoError(t, resp.Body.Close())
		})
	}
}

func TestNewCompressionMiddleware_Ignore(t *testing.T) {
	var ceh string // accept-encoding, content type

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	middleware, err := NewCompression(8, "/metrics")
	require.NoError(t, err)
	require.NotNil(t, middleware)

	// check if the route actually ignored
	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	require.NoError(t, err)
	req1.Header.Set("Accept-Encoding", "gzip")

	rc1 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc1, req1)

	ceh = rc1.Header().Get("Content-Encoding")
	assert.NotNil(t, ceh)
	assert.Empty(t, ceh)

	// check if other routes remains untouched
	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/alive", nil)
	require.NoError(t, err)
	req2.Header.Set("Accept-Encoding", "gzip")

	rc2 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc2, req2)

	ceh = rc2.Header().Get("Content-Encoding")
	assert.NotNil(t, ceh)
	assert.Equal(t, "gzip", ceh)
}

func TestNewCompressionMiddleware_Headers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware, err := NewCompression(8, "/metrics")
	require.NoError(t, err)

	tests := map[string]struct {
		cm               Func
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

	for encodingName, tt := range tests {
		t.Run(fmt.Sprintf("%q: compression middleware acts according the Accept-Encoding header", encodingName), func(t *testing.T) {
			require.NotNil(t, tt.cm)
			// given
			req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/alive", nil)
			require.NoError(t, err)
			if encodingName != "not present" {
				req1.Header.Set("Accept-Encoding", encodingName)
			}

			// when
			rc1 := httptest.NewRecorder()
			tt.cm(handler).ServeHTTP(rc1, req1)

			// then
			assert.Equal(t, tt.statusCode, rc1.Code)

			contentEncodingHeader := rc1.Header().Get("Content-Encoding")
			assert.NotNil(t, contentEncodingHeader)
			assert.Equal(t, tt.encodingExpected, contentEncodingHeader)
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

	for _, tt := range tests {
		t.Run(fmt.Sprintf("encoding %q is parsed as %s ; error is expected: %t ; %s", tt.given, tt.expected, tt.isErr, tt.optionalName), func(t *testing.T) {
			// when
			result, err := parseAcceptEncoding(tt.given)

			// then
			assert.Equal(t, tt.isErr, err != nil)
			assert.Equal(t, tt.expected, result)
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

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q check results in %t", tt.algorithm, tt.isSupported), func(t *testing.T) {
			// when
			result := !notSupportedCompression(tt.algorithm)

			// then
			assert.Equal(t, result, tt.isSupported)
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

	for _, tt := range tests {
		t.Run(fmt.Sprintf("for given priority: %q, expect %f", tt.priorityStr, tt.expected), func(t *testing.T) {
			// when
			result := parseWeight(tt.priorityStr)

			// then
			assert.Equal(t, tt.expected, result) //nolint: testifylint
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			selected, err := selectByWeight(tt.given)

			// then
			assert.Equal(t, tt.isErr, err != nil)
			assert.Equal(t, tt.expected, selected)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			addWithWeight(tt.weightedMap, tt.weight, tt.algorithm)

			// then
			assert.Equal(t, tt.expected, tt.weightedMap)
		})
	}
}

func TestIsConnectionReset(t *testing.T) {
	tests := map[string]struct {
		err      error
		expected bool
	}{
		"Broken pipe": {
			err:      errors.New("blah broken pipe blah"),
			expected: true,
		},
		"connection reset": {
			err:      errors.New("blah connection reset blah"),
			expected: true,
		},
		"read: connection reset": {
			err:      errors.New("blah read: connection reset blah"),
			expected: false,
		},
		"b00m random error": {
			err:      errors.New("blah b00m random error blah"),
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			result := isErrConnectionReset(tt.err)

			// then
			assert.Equal(t, tt.expected, result)
		})
	}
}

type failWriter struct{}

func (fw *failWriter) Header() http.Header {
	return http.Header{}
}

func (fw *failWriter) Write([]byte) (int, error) {
	return 0, errors.New("foo")
}

func (fw *failWriter) WriteHeader(_ int) {
}

func TestSetResponseWriterStatusOnResponseFailWrite(t *testing.T) {
	failWriter := &failWriter{}
	failDynamicCompressionResponseWriter := &dynamicCompressionResponseWriter{failWriter, "", nil, 0, 6}

	tests := []struct {
		Name           string
		ResponseWriter *responseWriter
	}{
		{
			Name:           "Failing responseWriter with http.responseWriter",
			ResponseWriter: newResponseWriter(failWriter, false),
		},
		{
			Name:           "Failing responseWriter with http.responseWriter",
			ResponseWriter: newResponseWriter(failDynamicCompressionResponseWriter, false),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			_, err := test.ResponseWriter.Write([]byte(`"foo":"bar"`))
			require.Error(t, err)
			assert.Equal(t, http.StatusOK, test.ResponseWriter.status)
		})
	}
}

func TestNewInjectObservability(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := NewInjectObservability()
	assert.NotNil(t, middleware)

	// check if the route actually ignored
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	require.NoError(t, err)

	rc := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc, req)

	assert.Equal(t, 200, rc.Code)
}

func TestNewCaching(t *testing.T) {
	tests := map[string]struct {
		cache       *httpcache.RouteCache
		expectedErr string
	}{
		"nil cache should return error":              {cache: nil, expectedErr: "route cache cannot be nil"},
		"non-nil cache should succeed without error": {cache: &httpcache.RouteCache{}, expectedErr: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cachingMiddleware, err := NewCaching(tt.cache)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, cachingMiddleware)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, cachingMiddleware)

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

			// check if the route is actually ignored
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
			require.NoError(t, err)

			rc := httptest.NewRecorder()
			cachingMiddleware(handler).ServeHTTP(rc, req)

			assert.Equal(t, 200, rc.Code)
		})
	}
}

func TestNewAppVersion(t *testing.T) {
	type args struct {
		name    string
		version string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"empty name should return error":                      {args: args{name: "", version: "version"}, expectedErr: "app name cannot be empty"},
		"empty version should return error":                   {args: args{name: "appName", version: ""}, expectedErr: "app version cannot be empty"},
		"valid name and version should succeed without error": {args: args{name: "name", version: "1.0"}, expectedErr: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			appNameVersionMiddleware, err := NewAppNameVersion(tt.args.name, tt.args.version)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, appNameVersionMiddleware)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, appNameVersionMiddleware)
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

			// check if the route actually ignored
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api", nil)
			require.NoError(t, err)
			rc := httptest.NewRecorder()

			appNameVersionMiddleware(handler).ServeHTTP(rc, req)

			assert.Equal(t, tt.args.version, rc.Header().Get(appVersionHeader))
			assert.Equal(t, tt.args.name, rc.Header().Get(appNameHeader))
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
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, getOrSetCorrelationID(tt.args.hdr))
			assert.NotEmpty(t, tt.args.hdr[correlation.HeaderID][0])
		})
	}
}
