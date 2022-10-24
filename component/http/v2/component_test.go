package v2

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHandler struct{}

func (s stubHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(200)
}

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		handler http.Handler
		oo      []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{
			handler: &stubHandler{},
			oo:      []OptionFunc{WithPort(50000)},
		}},
		"missing handler": {args: args{
			handler: nil,
			oo:      []OptionFunc{WithPort(50000)},
		}, expectedErr: "handler is nil"},
		"option error": {args: args{
			handler: &stubHandler{},
			oo:      []OptionFunc{WithPort(500000)},
		}, expectedErr: "invalid HTTP Port provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.handler, tt.args.oo...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestComponent_ListenAndServe_DefaultRoutes_Shutdown(t *testing.T) {
	listener, err := net.Listen("tcp", ":0") //nolint:gosec
	require.NoError(t, err)
	port, ok := listener.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	require.NoError(t, listener.Close())

	cmp, err := New(&stubHandler{}, WithPort(port.Port))
	assert.NoError(t, err)
	done := make(chan bool)
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		done <- true
	}()
	time.Sleep(10 * time.Millisecond)
	rsp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port.Port))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	cnl()
	assert.True(t, <-done)
}
