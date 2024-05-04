// Package mongo provides a client implementation for mongo with tracing and metrics included.
package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// TODO: Metrics???

// var cmdDurationMetrics *prometheus.HistogramVec

// func init() {
// 	cmdDurationMetrics = prometheus.NewHistogramVec(
// 		prometheus.HistogramOpts{
// 			Namespace: "client",
// 			Subsystem: "mongo",
// 			Name:      "cmd_duration_seconds",
// 			Help:      "Mongo commands completed by the client.",
// 		},
// 		[]string{"command", "success"},
// 	)
// 	prometheus.MustRegister(cmdDurationMetrics)
// }

// Connect with integrated observability via MongoDB's event package.
func Connect(ctx context.Context, oo ...*options.ClientOptions) (*mongo.Client, error) {
	clientOption := options.Client()
	clientOption.Monitor = otelmongo.NewMonitor()

	return mongo.Connect(ctx, append(oo, clientOption)...)
}
