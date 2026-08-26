//go:build integration

package sqs

import (
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	attributeKey = "key"
)

func TestSQSHelpersIntegration(t *testing.T) {
	t.Run("optional float attribute", func(t *testing.T) {
		got, err := getOptionalAttributeFloat64(map[string]string{attributeKey: "  "}, attributeKey)
		require.NoError(t, err)
		assert.InDelta(t, 0, got, 0)

		got, err = getOptionalAttributeFloat64(map[string]string{attributeKey: "12.5"}, attributeKey)
		require.NoError(t, err)
		assert.InDelta(t, 12.5, got, 0)

		_, err = getOptionalAttributeFloat64(map[string]string{attributeKey: "nope"}, attributeKey)
		require.ErrorContains(t, err, "could not convert nope to float64")
	})

	t.Run("correlation id", func(t *testing.T) {
		known := "known"
		assert.Equal(t, known, getCorrelationID(map[string]types.MessageAttributeValue{
			correlation.HeaderID: {StringValue: &known},
		}))

		got := getCorrelationID(map[string]types.MessageAttributeValue{
			correlation.HeaderID: {},
		})
		_, err := uuid.Parse(got)
		require.NoError(t, err)
	})
}
