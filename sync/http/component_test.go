package http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuilderWithoutOptions(t *testing.T) {
	got, err := NewBuilder().Create()
	assert.NotNil(t, got)
	assert.NoError(t, err)
}

func TestComponent_ListenAndServe_DefaultRoutes_Shutdown(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := NewBuilder().WithRoutes(rr).WithPort(50003).Create()
	assert.NoError(t, err)
	done := make(chan bool)
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, s.Run(ctx))
		done <- true
	}()
	time.Sleep(100 * time.Millisecond)
	assert.Len(t, s.routes, 15)
	cnl()
	assert.True(t, <-done)
}

func TestComponent_ListenAndServeTLS_DefaultRoutes_Shutdown(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := NewBuilder().WithRoutes(rr).WithSSL("testdata/server.pem", "testdata/server.key").WithPort(50003).Create()
	assert.NoError(t, err)
	done := make(chan bool)
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, s.Run(ctx))
		done <- true
	}()
	time.Sleep(100 * time.Millisecond)
	assert.Len(t, s.routes, 15)
	cnl()
	assert.True(t, <-done)
}

func TestComponent_ListenAndServeTLS_FailsInvalidCerts(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := NewBuilder().WithRoutes(rr).WithSSL("testdata/server.pem", "testdata/server.pem").Create()
	assert.NoError(t, err)
	assert.Error(t, s.Run(context.Background()))
}

func Test_createHTTPServer(t *testing.T) {
	cmp := Component{
		httpPort:         10000,
		httpReadTimeout:  5 * time.Second,
		httpWriteTimeout: 10 * time.Second,
	}
	s := cmp.createHTTPServer()
	assert.NotNil(t, s)
	assert.Equal(t, ":10000", s.Addr)
	assert.Equal(t, 5*time.Second, s.ReadTimeout)
	assert.Equal(t, 10*time.Second, s.WriteTimeout)
}

func Test_createHTTPServerUsingBuilder(t *testing.T) {

	var httpBuilderNoErrors = []error{}
	var httpBuilderAllErrors = []error{
		errors.New("Nil AliveCheckFunc was provided"),
		errors.New("Nil ReadyCheckFunc provided"),
		errors.New("Invalid HTTP Port provided"),
		errors.New("Negative or zero read timeout provided"),
		errors.New("Negative or zero write timeout provided"),
		errors.New("Empty Routes slice provided"),
		errors.New("Empty list of middlewares provided"),
		errors.New("Invalid cert or key provided"),
	}

	tests := map[string]struct {
		acf      AliveCheckFunc
		rcf      ReadyCheckFunc
		p        int
		rt       time.Duration
		wt       time.Duration
		rr       []Route
		mm       []MiddlewareFunc
		c        string
		k        string
		wantErrs []error
	}{
		"success": {
			acf: DefaultAliveCheck,
			rcf: DefaultReadyCheck,
			p:   httpPort,
			rt:  httpReadTimeout,
			wt:  httpIdleTimeout,
			rr: []Route{
				aliveCheckRoute(DefaultAliveCheck),
				readyCheckRoute(DefaultReadyCheck),
				metricRoute(),
			},
			mm: []MiddlewareFunc{
				NewRecoveryMiddleware(),
				panicMiddleware("error"),
			},
			c:        "cert.file",
			k:        "key.file",
			wantErrs: httpBuilderNoErrors,
		},
		"error in all builder steps": {
			acf:      nil,
			rcf:      nil,
			p:        -1,
			rt:       -10 * time.Second,
			wt:       -20 * time.Second,
			rr:       []Route{},
			mm:       []MiddlewareFunc{},
			c:        "",
			k:        "",
			wantErrs: httpBuilderAllErrors,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotHTTPComponent, gotErrs := NewBuilder().
				WithAliveCheckFunc(tc.acf).
				WithReadyCheckFunc(tc.rcf).
				WithPort(tc.p).
				WithReadTimeout(tc.rt).
				WithWriteTimeout(tc.wt).
				WithRoutes(tc.rr).
				WithMiddlewares(tc.mm...).
				WithSSL(tc.c, tc.k).
				Create()

			if len(tc.wantErrs) > 0 {
				assert.ObjectsAreEqual(tc.wantErrs, gotErrs)
				assert.Nil(t, gotHTTPComponent)
			} else {
				assert.NotNil(t, gotHTTPComponent)
				assert.IsType(t, &Component{}, gotHTTPComponent)
			}
		})
	}

}
