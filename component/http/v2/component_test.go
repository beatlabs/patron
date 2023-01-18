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
	hnd := &stubHandler{}
	type args struct {
		handler      http.Handler
		oo           []OptionFunc
		port         string
		readTimeout  string
		writeTimeout string
	}
	tests := map[string]struct {
		args        args
		expected    *Component
		expectedErr string
	}{
		"success": {
			args: args{handler: hnd},
			expected: &Component{
				port:                defaultPort,
				readTimeout:         defaultReadTimeout,
				writeTimeout:        defaultWriteTimeout,
				shutdownGracePeriod: defaultShutdownGracePeriod,
				handlerTimeout:      defaultHandlerTimeout,
				handler:             hnd,
			},
		},
		"success, env vars": {
			args: args{
				handler:      hnd,
				port:         "8080",
				readTimeout:  "10s",
				writeTimeout: "11s",
			},
			expected: &Component{
				port:                8080,
				readTimeout:         10 * time.Second,
				writeTimeout:        11 * time.Second,
				shutdownGracePeriod: defaultShutdownGracePeriod,
				handlerTimeout:      defaultHandlerTimeout,
				handler:             hnd,
			},
		},
		"failure, port env vars": {
			args: args{
				handler: hnd,
				port:    "aaa",
			},
			expectedErr: `env var for HTTP default port is not valid: strconv.ParseInt: parsing "aaa": invalid syntax`,
		},
		"failure, read timeout env vars": {
			args: args{
				handler:     hnd,
				readTimeout: "aaa",
			},
			expectedErr: `env var for HTTP read timeout is not valid: time: invalid duration "aaa"`,
		},
		"failure, write timeout env vars": {
			args: args{
				handler:      hnd,
				writeTimeout: "aaa",
			},
			expectedErr: `env var for HTTP write timeout is not valid: time: invalid duration "aaa"`,
		},
		"missing handler": {
			args:        args{handler: nil},
			expectedErr: "handler is nil",
		},
		"option error": {
			args:        args{handler: &stubHandler{}, oo: []OptionFunc{WithPort(500000)}},
			expectedErr: "invalid HTTP Port provided",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			if tt.args.port != "" {
				t.Setenv("PATRON_HTTP_DEFAULT_PORT", tt.args.port)
			}

			if tt.args.readTimeout != "" {
				t.Setenv("PATRON_HTTP_READ_TIMEOUT", tt.args.readTimeout)
			}

			if tt.args.writeTimeout != "" {
				t.Setenv("PATRON_HTTP_WRITE_TIMEOUT", tt.args.writeTimeout)
			}

			got, err := New(tt.args.handler, tt.args.oo...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
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
