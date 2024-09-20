package patron

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		t.Run(name, func(t *testing.T) {
			t.Setenv("PATRON_LOG_LEVEL", tt.lvl)
			assert.Equal(t, tt.want, getLogLevel())
		})
	}
}
