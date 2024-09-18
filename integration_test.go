//go:build integration

package patron

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServer_Run_Shutdown(t *testing.T) {
	tests := map[string]struct {
		cp      Component
		wantErr bool
	}{
		"success":       {cp: &testComponent{}, wantErr: false},
		"failed to run": {cp: &testComponent{errorRunning: true}, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				os.Clearenv()
			}()
			t.Setenv("PATRON_HTTP_DEFAULT_PORT", "50099")
			svc, err := New("test", "", WithJSONLogger())
			require.NoError(t, err)
			err = svc.Run(context.Background(), tt.cp)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServer_SetupTracing(t *testing.T) {
	tests := map[string]struct {
		cp      Component
		host    string
		port    string
		buckets string
	}{
		"success w/ empty tracing vars":     {cp: &testComponent{}},
		"success w/ empty tracing host":     {cp: &testComponent{}, port: "6831"},
		"success w/ empty tracing port":     {cp: &testComponent{}, host: "127.0.0.1"},
		"success":                           {cp: &testComponent{}, host: "127.0.0.1", port: "6831"},
		"success w/ custom default buckets": {cp: &testComponent{}, host: "127.0.0.1", port: "6831", buckets: ".1, .3"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.host != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_HOST", tt.host)
				require.NoError(t, err)
			}
			if tt.port != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_PORT", tt.port)
				require.NoError(t, err)
			}
			if tt.buckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", tt.buckets)
				require.NoError(t, err)
			}

			svc, err := New("test", "", WithJSONLogger())
			require.NoError(t, err)

			err = svc.Run(context.Background(), tt.cp)
			require.NoError(t, err)
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
