package kafka

import (
	"os"
	"testing"

	"go.uber.org/goleak"
)

// NOTE: Keep a single TestMain in this package. We only provide helper to run goleak here.
//
//nolint:testmain
func TestMain(m *testing.M) {
	if err := os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100"); err != nil {
		panic(err)
	}
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*client).backgroundMetadataUpdater"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*Broker).responseReceiver"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*asyncProducer).dispatcher"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*asyncProducer).retryHandler"),
		goleak.IgnoreTopFunction("github.com/beatlabs/patron/client/kafka.(*AsyncProducer).propagateError"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.newBrokerProducer.func1"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.newBrokerProducer.func2"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*brokerProducer).run"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*topicProducer).dispatch"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*partitionProducer).dispatch"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*syncProducer).handleSuccesses"),
		goleak.IgnoreTopFunction("github.com/IBM/sarama.(*syncProducer).handleErrors"),
		goleak.IgnoreTopFunction("github.com/rcrowley/go-metrics.(*meterArbiter).tick"),
	)
}
