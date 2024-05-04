package patron

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNew(t *testing.T) {
	httpBuilderAllErrors := "attributes are empty\nprovided WithSIGHUP handler was nil"

	tests := map[string]struct {
		name              string
		fields            []slog.Attr
		sighupHandler     func()
		observabilityConn *grpc.ClientConn
		uncompressedPaths []string
		wantErr           string
	}{
		"success": {
			name:              "name",
			fields:            []slog.Attr{slog.String("env", "dev")},
			sighupHandler:     func() { slog.Info("WithSIGHUP received: nothing setup") },
			observabilityConn: &grpc.ClientConn{},
			uncompressedPaths: []string{"/foo", "/bar"},
			wantErr:           "",
		},
		"name missing": {
			wantErr: "name is required",
		},
		"observability connection missing": {
			name:    "name",
			wantErr: "observability connection is required",
		},
		"nil inputs steps": {
			name:              "name",
			observabilityConn: &grpc.ClientConn{},
			wantErr:           httpBuilderAllErrors,
		},
		"error in all builder steps": {
			name:              "name",
			observabilityConn: &grpc.ClientConn{},
			uncompressedPaths: []string{},
			wantErr:           httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := New(tt.name, "1.0", tt.observabilityConn,
				WithLogFields(temp.fields...), WithJSONLogger(), WithSIGHUP(temp.sighupHandler))

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

			svc, err := New("test", "", &grpc.ClientConn{}, WithJSONLogger())

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
