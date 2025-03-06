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
			svc, err := New(context.Background(), "test", "", WithJSONLogger())
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

type testComponent struct {
	errorRunning bool
}

func (ts testComponent) Run(_ context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}
