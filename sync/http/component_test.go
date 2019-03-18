package http

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thebeatapp/patron/errors"
)

func ErrorOption() OptionFunc {
	return func(s *Component) error {
		return errors.New("TEST")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		options []OptionFunc
		wantErr bool
	}{
		{"success with no options", []OptionFunc{}, false},
		{"success with options", []OptionFunc{Port(50000)}, false},
		{"failed with error option", []OptionFunc{ErrorOption()}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestComponent_ListenAndServe_DefaultRoutes_Shutdown(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := New(Routes(rr), Port(50003))
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
	s, err := New(Routes(rr), Secure("testdata/server.pem", "testdata/server.key"), Port(50003))
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

func TestInfo(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := New(Routes(rr), Secure("testdata/server.pem", "testdata/server.key"), Port(50005))
	assert.NoError(t, err)
	expected := make(map[string]interface{})
	expected["type"] = "https"
	expected["port"] = 50005
	expected["read-timeout"] = httpReadTimeout.String()
	expected["write-timeout"] = httpWriteTimeout.String()
	expected["idle-timeout"] = httpIdleTimeout.String()
	expected["key-file"] = "testdata/server.key"
	expected["cert-file"] = "testdata/server.pem"
	assert.Equal(t, expected, s.Info())
}

func TestComponent_ListenAndServeTLS_FailsInvalidCerts(t *testing.T) {
	rr := []Route{NewRoute("/", "GET", nil, true, nil)}
	s, err := New(Routes(rr), Secure("testdata/server.pem", "testdata/server.pem"))
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
