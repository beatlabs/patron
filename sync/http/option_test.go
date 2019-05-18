package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"success", 30000, false},
		{"error for port number out of range", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Component{}
			err := Port(tt.port)(&s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.port, s.httpPort)
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetRoutes(t *testing.T) {
	tests := []struct {
		name    string
		rr      []Route
		wantErr bool
	}{
		{"success", []Route{NewGetRoute("/", testHandler{}.Process, true)}, false},
		{"error for no routes", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Component{}
			err := Routes(tt.rr)(&s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Len(t, s.routes, 1)
				assert.Equal(t, tt.rr[0].Method, s.routes[0].Method)
				assert.Equal(t, tt.rr[0].Pattern, s.routes[0].Pattern)
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetMiddlewares(t *testing.T) {
	tests := []struct {
		name    string
		mm      []MiddlewareFunc
		wantErr bool
	}{
		{"success", []MiddlewareFunc{func(next http.HandlerFunc) http.HandlerFunc { return next }}, false},
		{"error for empty middlewares", []MiddlewareFunc{}, true},
		{"error for nil middlewares", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Component{}
			err := Middlewares(tt.mm...)(&s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Len(t, s.middlewares, 1)
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetHealthCheck(t *testing.T) {
	tests := []struct {
		name    string
		hcf     HealthCheckFunc
		wantErr bool
	}{
		{"success", func() HealthStatus { return Healthy }, false},
		{"error for no routes", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Component{}
			err := HealthCheck(tt.hcf)(&s)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, s.hc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s.hc)
			}
		})
	}
}

func TestSetSecure(t *testing.T) {
	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{"success", "certFile", "keyFile", false},
		{"failed missing cert file", "", "keyFile", true},
		{"failed missing key file", "certFile", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Component{}
			err := Secure(tt.certFile, tt.keyFile)(&s)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, s.certFile)
				assert.Empty(t, s.keyFile)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, s.certFile)
				assert.NotEmpty(t, s.keyFile)
			}
		})
	}
}

func TestTimeouts(t *testing.T) {
	c := Component{}
	err := Timeouts(2*time.Second, 3*time.Second)(&c)
	assert.NoError(t, err)
	assert.Equal(t, 2*time.Second, c.httpReadTimeout)
	assert.Equal(t, 3*time.Second, c.httpWriteTimeout)
}
