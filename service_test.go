package patron

import (
	"context"
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
			ctx := context.Background()
			gotService, gotErr := New(ctx, tt.name, "1.0",
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
