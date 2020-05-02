package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestCachingMiddleware(t *testing.T) {

	getRequest, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	postRequest, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}

	testingCache := newTestingCache()
	testingCache.instant = now

	tests := []struct {
		name         string
		args         args
		r            *http.Request
		expectedCode int
		expectedBody string
		cacheState   cacheState
	}{
		{"caching middleware with POST request", args{next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(202)
			i, err := w.Write([]byte{1, 2, 3, 4})
			assert.NoError(t, err)
			assert.Equal(t, 4, i)
		}), mws: []MiddlewareFunc{NewCachingMiddleware(NewRouteCache(testingCache, Age{Max: 1 * time.Second}))}},
			postRequest, 202, "\x01\x02\x03\x04", cacheState{}},
		{"caching middleware with GET request", args{next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			i, err := w.Write([]byte{1, 2, 3, 4})
			assert.NoError(t, err)
			assert.Equal(t, 4, i)
		}), mws: []MiddlewareFunc{NewCachingMiddleware(NewRouteCache(testingCache, Age{Max: 1 * time.Second}))}},
			getRequest, 200, "\x01\x02\x03\x04", cacheState{
				setOps: 1,
				getOps: 1,
				size:   1,
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, tt.r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
			assertCacheState(t, *testingCache, tt.cacheState)
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
