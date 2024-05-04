//go:build integration
// +build integration

package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TODO: assert metrics and traces

func TestConnectAndExecute(t *testing.T) {
	client, err := Connect(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("success", func(t *testing.T) {
		err = client.Ping(context.Background(), nil)
		require.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		names, err := client.ListDatabaseNames(context.Background(), bson.M{})
		assert.Error(t, err)
		assert.Empty(t, names)
	})
}
