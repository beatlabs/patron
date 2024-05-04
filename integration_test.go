//go:build integration
// +build integration

package patron

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer_Run_Shutdown(t *testing.T) {
	conn, err := grpc.NewClient("localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	tests := map[string]struct {
		cp      Component
		wantErr bool
	}{
		"success":       {cp: &testComponent{}, wantErr: false},
		"failed to run": {cp: &testComponent{errorRunning: true}, wantErr: true},
	}
	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			defer func() {
				os.Clearenv()
			}()
			t.Setenv("PATRON_HTTP_DEFAULT_PORT", "50099")
			svc, err := New("test", "", conn, WithJSONLogger())
			assert.NoError(t, err)
			err = svc.Run(context.Background(), tt.cp)
			if temp.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_SetupTracing(t *testing.T) {
	conn, err := grpc.NewClient("localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	tests := []struct {
		name    string
		cp      Component
		host    string
		port    string
		buckets string
	}{
		{name: "success w/ empty tracing vars", cp: &testComponent{}},
		{name: "success w/ empty tracing host", cp: &testComponent{}, port: "6831"},
		{name: "success w/ empty tracing port", cp: &testComponent{}, host: "127.0.0.1"},
		{name: "success", cp: &testComponent{}, host: "127.0.0.1", port: "6831"},
		{name: "success w/ custom default buckets", cp: &testComponent{}, host: "127.0.0.1", port: "6831", buckets: ".1, .3"},
	}
	for _, tt := range tests {
		temp := tt
		t.Run(temp.name, func(t *testing.T) {
			defer os.Clearenv()

			if temp.host != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_HOST", temp.host)
				assert.NoError(t, err)
			}
			if temp.port != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_PORT", temp.port)
				assert.NoError(t, err)
			}
			if temp.buckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", temp.buckets)
				assert.NoError(t, err)
			}

			svc, err := New("test", "", conn, WithJSONLogger())
			assert.NoError(t, err)

			err = svc.Run(context.Background(), tt.cp)
			assert.NoError(t, err)
		})
	}
}

type testComponent struct {
	errorRunning bool
}

func (ts testComponent) Run(_ context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}
