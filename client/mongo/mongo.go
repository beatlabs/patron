// Package mongo provides a client implementation for mongo with tracing and metrics included.
package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

// Connect with integrated observability via MongoDB's event package.
func Connect(ctx context.Context, oo ...*options.ClientOptions) (*mongo.Client, error) {
	clientOption := options.Client()
	clientOption.SetMonitor(newObservabilityMonitor(otelmongo.NewMonitor()))

	return mongo.Connect(ctx, append(oo, clientOption)...)
}
