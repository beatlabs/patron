package patron

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type stubComponent struct{}

func (stubComponent) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
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

func TestService_Run_RejectsNilComponentAtAnyIndex(t *testing.T) {
	svc, err := New("test", "1.0", WithJSONLogger())
	require.NoError(t, err)

	tests := map[string]struct {
		components []Component
		wantErr    string
	}{
		"empty":           {components: nil, wantErr: "components are empty or nil"},
		"nil at index 0":  {components: []Component{nil}, wantErr: "component at index 0 is nil"},
		"nil at index 1":  {components: []Component{stubComponent{}, nil}, wantErr: "component at index 1 is nil"},
		"nil at index 2":  {components: []Component{stubComponent{}, stubComponent{}, nil}, wantErr: "component at index 2 is nil"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := svc.Run(context.Background(), tt.components...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
