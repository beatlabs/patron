package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
			//next
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
			rw := newResponseWriter(rc)
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
		{"tracing middleware", args{next: handler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, 202, ""},
		{"recovery middleware from panic 1", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware("error")}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 2", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(errors.New("error"))}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 3", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(-1)}}, 500, "Internal Server Error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
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
		{"tracing middleware - error", args{next: errorHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, http.StatusInternalServerError, "foo", "foo"},
		{"tracing middleware - success", args{next: successHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, http.StatusOK, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mtr.Reset()
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
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
	rw := newResponseWriter(rc)

	_, err := rw.Write([]byte("test"))
	assert.NoError(t, err)
	rw.WriteHeader(202)

	assert.Equal(t, 202, rw.status, "status expected 202 but got %d", rw.status)
	assert.Len(t, rw.Header(), 1, "Header count expected to be 1")
	assert.True(t, rw.statusHeaderWritten, "expected to be true")
	assert.Equal(t, "test", rc.Body.String(), "body expected to be test but was %s", rc.Body.String())
}

func TestStripQueryString(t *testing.T) {
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
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
			req, err := http.NewRequest("GET", "/test", nil)
			assert.NoError(t, err)

			req.Header.Set("Accept-Encoding", name)
			compressionMiddleware := tc.cm
			assert.NoError(t, err)
			assert.NotNil(t, compressionMiddleware)

			rc := httptest.NewRecorder()
			compressionMiddleware(handler).ServeHTTP(rc, req)
			actual := rc.Header().Get("Content-Encoding")
			assert.NotNil(t, actual)
			assert.Equal(t, name, actual)
		})
	}
}

func TestNewCompressionMiddleware_Ignore(t *testing.T) {
	var ceh, cth string // accept-encoding, content type

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

	cth = rc1.Header().Get("Content-Type")
	assert.NotNil(t, cth)
	assert.Equal(t, cth, "")

	// check if other routes remains untouched
	req2, err := http.NewRequest("GET", "/alive", nil)
	assert.NoError(t, err)
	req2.Header.Set("Accept-Encoding", "gzip")
	req2.Header.Set("Content-Type", "application/json")

	rc2 := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rc2, req2)

	ceh = rc2.Header().Get("Content-Encoding")
	assert.NotNil(t, ceh)
	assert.Equal(t, "gzip", ceh)

	cth = rc2.Header().Get("Content-Type")
	assert.NotNil(t, cth)
	assert.Equal(t, "application/json", cth)
}

func TestNewCompressionMiddleware_Headers(t *testing.T) {
	var ceh, cth string // accept-encoding, content type

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })

	tests := map[string]struct {
		cm MiddlewareFunc
	}{
		"gzip":    {cm: NewCompressionMiddleware(8, "/metrics")},
		"deflate": {cm: NewCompressionMiddleware(8, "/metrics")},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			middleware := tc.cm
			assert.NotNil(t, middleware)

			// check if the route actually ignored
			req1, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			req1.Header.Set("Accept-Encoding", name)
			req1.Header.Set("Content-Type", "text/plain")

			rc1 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc1, req1)

			ceh = rc1.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, name, ceh)

			cth = rc1.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, "text/plain", cth)

			// check if other routes remains untouched
			req2, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			req2.Header.Set("Accept-Encoding", name)
			req2.Header.Set("Content-Type", "application/json")

			rc2 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc2, req2)

			ceh = rc2.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, name, ceh)

			cth = rc2.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, "application/json", cth)
		})
	}
}

func TestSelectEncoding(t *testing.T) {
	testFn := func(hdr string, expected string, isErr bool) func(t *testing.T) {
		return func(t *testing.T) {
			result, err := parseAcceptEncoding(hdr)
			assert.Equal(t, isErr, err != nil)
			assert.Equal(t, expected, result)
		}
	}

	t.Run("*", testFn("*", "*", false))
	t.Run("gzip", testFn("gzip", "gzip", false))
	t.Run("deflate", testFn("deflate", "deflate", false))

	t.Run("whatever, not supported", testFn("whatever", "", true))
	t.Run("whatever, but also a star", testFn("whatever, *", "*", false))

	t.Run("gzip, deflate", testFn("gzip, deflate", "gzip", false))
	t.Run("whatever, gzip, deflate", testFn("whatever, gzip, deflate", "gzip", false))
	t.Run("gzip, whatever, deflate", testFn("gzip, whatever, deflate", "gzip", false))
	t.Run("gzip, deflate, whatever", testFn("gzip, deflate, whatever", "gzip", false))

	t.Run("gzip,deflate", testFn("gzip,deflate", "gzip", false))
	t.Run("gzip,whatever,deflate", testFn("gzip,whatever,deflate", "gzip", false))
	t.Run("whatever,gzip,deflate", testFn("whatever,gzip,deflate", "gzip", false))
	t.Run("gzip,deflate,whatever", testFn("gzip,deflate,whatever", "gzip", false))

	t.Run("deflate, gzip", testFn("deflate, gzip", "deflate", false))
	t.Run("whatever, deflate, gzip", testFn("whatever, deflate, gzip", "deflate", false))
	t.Run("deflate, whatever, gzip", testFn("deflate, whatever, gzip", "deflate", false))
	t.Run("deflate, gzip, whatever", testFn("deflate, gzip, whatever", "deflate", false))

	t.Run("deflate,gzip", testFn("deflate, gzip", "deflate", false))
	t.Run("whatever,deflate,gzip", testFn("whatever,deflate,gzip", "deflate", false))
	t.Run("deflate,whatever,gzip", testFn("deflate,whatever,gzip", "deflate", false))
	t.Run("deflate,gzip,whatever", testFn("deflate,gzip,whatever", "deflate", false))

	t.Run("equal weights", testFn("gzip;q=1.0, deflate;q=1.0", "gzip", false))
	t.Run("equal weights 2", testFn("deflate;q=1.0, gzip;q=1.0", "deflate", false))

	t.Run("gzip;q=1.0, deflate;q=0.5", testFn("gzip;q=1.0, deflate;q=0.5", "gzip", false))
	t.Run("gzip;q=1.0, deflate;q=0.5, *;q=0.2", testFn("gzip;q=1.0, deflate;q=0.5, *;q=0.2", "gzip", false))
	t.Run("deflate;q=1.0, gzip;q=0.5", testFn("deflate;q=1.0, gzip;q=0.5", "deflate", false))
	t.Run("deflate;q=1.0, gzip;q=0.5, *;q=0.2", testFn("deflate;q=1.0, gzip;q=0.5, *;q=0.2", "deflate", false))

	t.Run("gzip;q=0.5, deflate;q=1.0", testFn("gzip;q=0.5, deflate;q=1.0", "deflate", false))
	t.Run("gzip;q=0.5, deflate;q=1.0, *;q=0.2", testFn("gzip;q=0.5, deflate;q=1.0, *;q=0.2", "deflate", false))
	t.Run("deflate;q=0.5, gzip;q=1.0", testFn("deflate;q=0.5, gzip;q=1.0", "gzip", false))
	t.Run("deflate;q=0.5, gzip;q=1.0, *;q=0.2", testFn("deflate;q=0.5, gzip;q=1.0, *;q=0.2", "gzip", false))

	t.Run("whatever;q=1.0, *;q=0.2", testFn("whatever;q=1.0, *;q=0.2", "*", false))

	t.Run("deflate, gzip;q=1.0", testFn("deflate, gzip;q=1.0", "gzip", false))
	t.Run("deflate;q=0.5, gzip", testFn("deflate;q=0.5, gzip", "deflate", false))
}

func TestSupported(t *testing.T) {
	testFn := func(algorithm string, expected bool) func(t *testing.T) {
		return func(t *testing.T) {
			// when
			notSupported := notSupportedCompression(algorithm)

			// then
			assert.Equal(t, notSupported, !expected)
		}
	}

	t.Run("gzip", testFn("gzip", true))
	t.Run("deflate", testFn("deflate", true))
	t.Run("star", testFn("*", true))
	t.Run("something else", testFn("something else", false))
}

func TestParseWeights(t *testing.T) {
	testFn := func(qStr string, expected float64, isErr bool) func(t *testing.T) {
		return func(t *testing.T) {
			// when
			result, err := parseWeight(qStr)

			// then
			assert.Equal(t, isErr, err != nil)
			assert.Equal(t, expected, result)
		}
	}

	t.Run("q=1.0", testFn("q=1.0", 1.0, false))
	t.Run("q=0.5", testFn("q=0.5", 0.5, false))
	t.Run("q=", testFn("q=", 0.0, true))
	t.Run("empty string", testFn("", 0.0, true))
}

func TestSelectByWeight(t *testing.T) {
	testFn := func(given map[float64]string, expected string, isErr bool) func(t *testing.T) {
		return func(t *testing.T) {
			// when
			selected, err := selectByWeight(given)

			// then
			assert.Equal(t, isErr, err != nil)
			assert.Equal(t, expected, selected)
		}
	}

	t.Run("sorted", testFn(map[float64]string{1.0: "gzip", 0.5: "deflate"}, "gzip", false))
	t.Run("not sorted", testFn(map[float64]string{0.5: "gzip", 1.0: "deflate"}, "deflate", false))
	t.Run("empty", testFn(map[float64]string{}, "", true))
}

func TestAddWithWeight(t *testing.T) {
	testFn := func(
		weightedMap map[float64]string, weight float64, algorithm string,
		expected map[float64]string) func(t *testing.T) {
		return func(t *testing.T) {
			// given

			// when
			addWithWeight(weightedMap, weight, algorithm)

			// then
			assert.Equal(t, expected, weightedMap)
		}
	}

	t.Run("empty", testFn(map[float64]string{}, 1.0, "gzip", map[float64]string{1.0: "gzip"}))
	t.Run("new", testFn(map[float64]string{1.0: "gzip"}, 0.5, "deflate", map[float64]string{1.0: "gzip", 0.5: "deflate"}))
	t.Run("already exists", testFn(map[float64]string{1.0: "gzip"}, 1.0, "deflate", map[float64]string{1.0: "gzip"}))
}
