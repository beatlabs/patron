package http

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	errs "github.com/beatlabs/patron/errors"
	"github.com/stretchr/testify/assert"
)

func TestBuilderWithoutOptions(t *testing.T) {
	got, err := NewBuilder().Create()
	assert.NotNil(t, got)
	assert.NoError(t, err)
}

func TestComponent_ListenAndServe_DefaultRoutes_Shutdown(t *testing.T) {
	rb := NewRoutesBuilder().
		Append(NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodGet().WithTrace())
	s, err := NewBuilder().WithRoutesBuilder(rb).WithPort(50013).Create()
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
	rb := NewRoutesBuilder().Append(NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodGet())
	s, err := NewBuilder().WithRoutesBuilder(rb).WithSSL("testdata/server.pem", "testdata/server.key").WithPort(50014).Create()
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
	rb := NewRoutesBuilder().Append(NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodGet())
	s, err := NewBuilder().WithRoutesBuilder(rb).WithSSL("testdata/server.pem", "testdata/server.pem").Create()
	assert.NoError(t, err)
	assert.Error(t, s.Run(context.Background()))
}

func Test_createHTTPServer(t *testing.T) {
	cmp := Component{
		httpPort:         10000,
		httpReadTimeout:  5 * time.Second,
		httpWriteTimeout: 10 * time.Second,
	}
	ctx := context.Background()
	s := cmp.createHTTPServer(ctx)
	assert.NotNil(t, s)
	assert.Equal(t, ":10000", s.Addr)
	assert.Equal(t, 5*time.Second, s.ReadTimeout)
	assert.Equal(t, 10*time.Second, s.WriteTimeout)
}

func Test_createHTTPServerUsingBuilder(t *testing.T) {

	var httpBuilderNoErrors = []error{}
	var httpBuilderAllErrors = []error{
		errors.New("nil AliveCheckFunc was provided"),
		errors.New("nil ReadyCheckFunc provided"),
		errors.New("invalid HTTP Port provided"),
		errors.New("negative or zero read timeout provided"),
		errors.New("negative or zero write timeout provided"),
		errors.New("route builder is nil"),
		errors.New("empty list of middlewares provided"),
		errors.New("invalid cert or key provided"),
	}

	rb := NewRoutesBuilder().Append(NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodGet())

	tests := map[string]struct {
		acf      AliveCheckFunc
		rcf      ReadyCheckFunc
		p        int
		rt       time.Duration
		wt       time.Duration
		rb       *RoutesBuilder
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
			rb:  rb,
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
			rb:       nil,
			mm:       []MiddlewareFunc{},
			c:        "",
			k:        "",
			wantErrs: httpBuilderAllErrors,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotHTTPComponent, err := NewBuilder().WithAliveCheckFunc(tc.acf).WithReadyCheckFunc(tc.rcf).
				WithPort(tc.p).WithReadTimeout(tc.rt).WithWriteTimeout(tc.wt).WithRoutesBuilder(tc.rb).
				WithMiddlewares(tc.mm...).WithSSL(tc.c, tc.k).Create()

			if len(tc.wantErrs) > 0 {
				assert.EqualError(t, err, errs.Aggregate(tc.wantErrs...).Error())
				assert.Nil(t, gotHTTPComponent)
			} else {
				assert.NotNil(t, gotHTTPComponent)
				assert.IsType(t, &Component{}, gotHTTPComponent)
			}
		})
	}
}
