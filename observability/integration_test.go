//go:build integration

package observability

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/observability/log"
	"github.com/stretchr/testify/require"
)

func TestSetup(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	ctx := context.Background()

	got, err := Setup(ctx, Config{
		LogConfig: log.Config{
			Level: "debug",
		},
	})
	require.NoError(t, err)

	require.NoError(t, got.Shutdown(ctx))
}
