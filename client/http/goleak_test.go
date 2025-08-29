package http

import (
	"os"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	if err := os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100"); err != nil {
		panic(err)
	}
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("net/http.setRequestCancel.func4"),
	)
}
