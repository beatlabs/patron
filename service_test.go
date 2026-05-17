package patron

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"

	"github.com/beatlabs/patron/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type testRunComponent struct {
	ran atomic.Bool
	err error
}

func (c *testRunComponent) Run(_ context.Context) error {
	c.ran.Store(true)
	return c.err
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/metric.(*PeriodicReader).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
	)
}

func TestNew(t *testing.T) {
	httpBuilderAllErrors := "attributes are empty\nprovided WithSIGHUP handler was nil"

	tests := map[string]struct {
		name          string
		fields        []slog.Attr
		sighupHandler func()

		uncompressedPaths []string
		wantErr           string
	}{
		"success": {
			name:              "name",
			fields:            []slog.Attr{slog.String("env", "dev")},
			sighupHandler:     func() { slog.Info("WithSIGHUP received: nothing setup") },
			uncompressedPaths: []string{"/foo", "/bar"},
			wantErr:           "",
		},
		"name missing": {
			wantErr: "name is required",
		},
		"nil inputs steps": {
			name:    "name",
			wantErr: httpBuilderAllErrors,
		},
		"error in all builder steps": {
			name:              "name",
			uncompressedPaths: []string{},
			wantErr:           httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := New(tt.name, "1.0",
				WithLogFields(tt.fields...), WithJSONLogger(), WithSIGHUP(tt.sighupHandler))

			if tt.wantErr != "" {
				require.EqualError(t, gotErr, tt.wantErr)
				assert.Nil(t, gotService)
			} else {
				require.NoError(t, gotErr)
				assert.NotNil(t, gotService)
				assert.IsType(t, &Service{}, gotService)
				assert.NotNil(t, gotService.termSig)
				assert.NotNil(t, gotService.sighupHandler)
			}
		})
	}
}

func TestRunRejectsNilComponentBeforeStartingAny(t *testing.T) {
	t.Parallel()

	service := &Service{
		name:                  "test-service",
		termSig:               make(chan os.Signal, 1),
		observabilityProvider: &observability.Provider{},
	}
	component := &testRunComponent{}

	err := service.Run(context.Background(), component, nil)

	require.EqualError(t, err, "components are empty or nil")
	assert.False(t, component.ran.Load())
}

func TestRunRejectsEmptyComponents(t *testing.T) {
	t.Parallel()

	service := &Service{
		name:                  "test-service",
		termSig:               make(chan os.Signal, 1),
		observabilityProvider: &observability.Provider{},
	}

	err := service.Run(context.Background())

	require.EqualError(t, err, "components are empty or nil")
}

func TestNew_WithJSONLogger_ConfiguresDefaultLogger(t *testing.T) {
	output := captureStderr(t, func() {
		svc, err := New("name", "1.0", WithJSONLogger())
		require.NoError(t, err)
		require.NotNil(t, svc)

		slog.Info("json logger configured")
	})

	var payload map[string]any
	require.NoError(t, json.Unmarshal(output, &payload))
	assert.Equal(t, "json logger configured", payload["msg"])
}

func TestNew_WithLogFields_ConfiguresDefaultLoggerAttributes(t *testing.T) {
	output := captureStderr(t, func() {
		svc, err := New("name", "1.0", WithLogFields(slog.String("env", "dev")))
		require.NoError(t, err)
		require.NotNil(t, svc)

		slog.Info("log fields configured")
	})

	assert.Contains(t, string(output), "env=dev")
	assert.Contains(t, string(output), "msg=\"log fields configured\"")
}

func TestNew_WithJSONLoggerAndLogFields_ConfiguresDefaultLogger(t *testing.T) {
	output := captureStderr(t, func() {
		svc, err := New("name", "1.0", WithJSONLogger(), WithLogFields(slog.String("env", "dev")))
		require.NoError(t, err)
		require.NotNil(t, svc)

		slog.Info("json logger with fields configured")
	})

	var payload map[string]any
	require.NoError(t, json.Unmarshal(output, &payload))
	assert.Equal(t, "json logger with fields configured", payload["msg"])
	assert.Equal(t, "dev", payload["env"])
}

func captureStderr(t *testing.T, fn func()) []byte {
	t.Helper()

	originalStderr := os.Stderr
	originalLogger := slog.Default()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() {
		os.Stderr = originalStderr
		slog.SetDefault(originalLogger)
	}()

	os.Stderr = w

	fn()
	require.NoError(t, w.Close())

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return bytes.TrimSpace(buf.Bytes())
}
