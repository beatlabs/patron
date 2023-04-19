package patron

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestNew(t *testing.T) {
	httpBuilderAllErrors := "attributes are empty\nprovided WithSIGHUP handler was nil\n"

	tests := map[string]struct {
		name              string
		fields            []slog.Attr
		sighupHandler     func()
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
			sighupHandler:     nil,
			uncompressedPaths: nil,
			wantErr:           "name is required",
		},
		"nil inputs steps": {
			name:              "name",
			sighupHandler:     nil,
			uncompressedPaths: nil,
			wantErr:           httpBuilderAllErrors,
		},
		"error in all builder steps": {
			name:              "name",
			sighupHandler:     nil,
			uncompressedPaths: []string{},
			wantErr:           httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := New(tt.name, "1.0", WithLogFields(temp.fields...), WithJSONLogger(),
				WithSIGHUP(temp.sighupHandler))

			if temp.wantErr != "" {
				assert.EqualError(t, gotErr, temp.wantErr)
				assert.Nil(t, gotService)
			} else {
				assert.Nil(t, gotErr)
				assert.NotNil(t, gotService)
				assert.IsType(t, &Service{}, gotService)
				assert.NotNil(t, gotService.termSig)
				assert.NotNil(t, gotService.sighupHandler)
			}
		})
	}
}

func TestServer_Run_Shutdown(t *testing.T) {
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
			svc, err := New("test", "", WithJSONLogger())
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

			svc, err := New("test", "", WithJSONLogger())
			assert.NoError(t, err)

			err = svc.Run(context.Background(), tt.cp)
			assert.NoError(t, err)
		})
	}
}

func TestNewServer_FailingConditions(t *testing.T) {
	tests := map[string]struct {
		jaegerSamplerParam       string
		jaegerBuckets            string
		expectedConstructorError string
	}{
		"failure w/ sampler param":             {jaegerSamplerParam: "foo", expectedConstructorError: "env var for jaeger sampler param is not valid: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
		"failure w/ overflowing sampler param": {jaegerSamplerParam: "8", expectedConstructorError: "cannot initialize jaeger tracer: invalid Param for probabilistic sampler; expecting value between 0 and 1, received 8"},
		"failure w/ custom default buckets":    {jaegerSamplerParam: "1", jaegerBuckets: "foo", expectedConstructorError: "env var for jaeger default buckets contains invalid value: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.jaegerSamplerParam != "" {
				err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", tt.jaegerSamplerParam)
				require.NoError(t, err)
			}
			if tt.jaegerBuckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", tt.jaegerBuckets)
				require.NoError(t, err)
			}

			svc, err := New("test", "", WithJSONLogger())

			if tt.expectedConstructorError != "" {
				require.EqualError(t, err, tt.expectedConstructorError)
				require.Nil(t, svc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, svc)

			// start running with a canceled context, on purpose
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = svc.Run(ctx)
			require.NoError(t, err)

			require.Equal(t, err, context.Canceled)
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

func Test_getLogLevel(t *testing.T) {
	tests := map[string]struct {
		lvl  string
		want slog.Level
	}{
		"debug":         {lvl: "debug", want: slog.LevelDebug},
		"info":          {lvl: "info", want: slog.LevelInfo},
		"warn":          {lvl: "warn", want: slog.LevelWarn},
		"error":         {lvl: "error", want: slog.LevelError},
		"invalid level": {lvl: "invalid", want: slog.LevelInfo},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Setenv("PATRON_LOG_LEVEL", tt.lvl)
			assert.Equal(t, tt.want, getLogLevel())
		})
	}
}
