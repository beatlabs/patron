package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thebeatapp/patron/errors"
	"github.com/thebeatapp/patron/sync/http/auth"
)

func testHandle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(202)
}

func testPanicHandleString(w http.ResponseWriter, r *http.Request) {
	panic("test")
}

func testPanicHandleError(w http.ResponseWriter, r *http.Request) {
	panic(errors.New("TEST"))
}

func testPanicHandleInt(w http.ResponseWriter, r *http.Request) {
	panic(1000)
}

// A middleware generator that tags resp for assertions
func tagMiddleware(tag string) MiddlewareFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(tag))
			h(w, r)
		}
	}
}

func TestMiddlewareDefaults(t *testing.T) {
	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next  http.HandlerFunc
		trace bool
		auth  auth.Authenticator
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
	}{
		{"middleware success", args{next: testHandle, trace: false, auth: &MockAuthenticator{success: true}}, 202},
		{"middleware trace success", args{next: testHandle, trace: true, auth: &MockAuthenticator{success: true}}, 202},
		{"middleware panic string", args{next: testPanicHandleString, trace: true, auth: &MockAuthenticator{success: true}}, 500},
		{"middleware panic error", args{next: testPanicHandleError, trace: true, auth: &MockAuthenticator{success: true}}, 500},
		{"middleware panic other", args{next: testPanicHandleInt, trace: true, auth: &MockAuthenticator{success: true}}, 500},
		{"middleware auth error", args{next: testPanicHandleInt, trace: true, auth: &MockAuthenticator{err: errors.New("TEST")}}, 500},
		{"middleware auth failure", args{next: testPanicHandleInt, trace: true, auth: &MockAuthenticator{success: false}}, 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			MiddlewareDefaults(tt.args.trace, tt.args.auth, "path", tt.args.next)(resp, r)
			assert.Equal(t, tt.expectedCode, resp.Code)
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	type args struct {
		next http.HandlerFunc
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"middleware 1,2,3 and finish", args{next: testHandle, mws: []MiddlewareFunc{t1, t2, t3}}, 202, "t1\nt2\nt3\n"},
		{"middleware 1,2 and finish", args{next: testHandle, mws: []MiddlewareFunc{t1, t2}}, 202, "t1\nt2\n"},
		{"no middleware and finish", args{next: testHandle, mws: []MiddlewareFunc{}}, 202, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
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
	assert.Len(t, rw.Header(), 1, "header count expected to be 1")
	assert.True(t, rw.statusHeaderWritten, "expected to be true")
	assert.Equal(t, "test", rc.Body.String(), "body expected to be test but was %s", rc.Body.String())
}

// func Test_authMiddleware(t *testing.T) {
// 	r, err := http.NewRequest("POST", "/test", nil)
// 	assert.NoError(t, err)

// 	type args struct {
// 		auth Authenticator
// 		next http.HandlerFunc
// 		resp *httptest.ResponseRecorder
// 	}
// 	tests := []struct {
// 		name         string
// 		args         args
// 		expectedCode int
// 	}{
// 		{name: "authenticated", args: args{auth: &MockAuthenticator{success: true}}, expectedCode: 202},
// 		{name: "unauthorized", args: args{auth: &MockAuthenticator{success: false}}, expectedCode: 401},
// 		{name: "error", args: args{auth: &MockAuthenticator{err: errors.New("TEST")}}, expectedCode: 500},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			authMiddleware(tt.args.auth, testHandle)(tt.args.resp, r)
// 			assert.Equal(t, tt.expectedCode, tt.args.resp.Code)
// 		})
// 	}
// }
