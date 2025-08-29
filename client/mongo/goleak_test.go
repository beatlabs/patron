package mongo

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
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*pool).createConnections.func2"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*pool).maintain"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*cancellListener).Listen"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*rttMonitor).runHellos"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*connection).read"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*Server).update"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.(*Server).check"),
		goleak.IgnoreTopFunction("go.mongodb.org/mongo-driver/x/mongo/driver/topology.initConnection.ReadWireMessage"),
	)
}
