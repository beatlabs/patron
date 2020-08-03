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
