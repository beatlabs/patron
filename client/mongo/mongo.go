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
	monitor, err := newObservabilityMonitor(otelmongo.NewMonitor())
	if err != nil {
		return nil, err
	}
	clientOption := options.Client()
	clientOption.SetMonitor(monitor)

	return mongo.Connect(ctx, append(oo, clientOption)...)
}
