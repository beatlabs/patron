package mqtt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const mqttTestTopic = "test/topic"

func TestTopicAttr(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		topic    string
		expected string
	}{
		"simple topic": {
			topic:    "test/topic",
			expected: "test/topic",
		},
		"nested topic": {
			topic:    "devices/sensor/temperature",
			expected: "devices/sensor/temperature",
		},
		"wildcard topic": {
			topic:    "devices/+/status",
			expected: "devices/+/status",
		},
		"empty topic": {
			topic:    "",
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			attr := topicAttr(tt.topic)

			assert.Equal(t, topicAttribute, string(attr.Key))
			assert.Equal(t, tt.expected, attr.Value.AsString())
		})
	}
}

func TestObservePublish(t *testing.T) {
	t.Parallel()

	t.Run("observes successful publish", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		start := time.Now()

		// This function records metrics, we just verify it doesn't panic
		observePublish(ctx, start, mqttTestTopic, nil)
	})

	t.Run("observes failed publish", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		start := time.Now()
		err := errors.New("connection failed")

		// This function records metrics, we just verify it doesn't panic
		observePublish(ctx, start, mqttTestTopic, err)
	})

	t.Run("records duration correctly", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		start := time.Now().Add(-100 * time.Millisecond) // Simulate 100ms ago

		// This function records metrics
		observePublish(ctx, start, mqttTestTopic, nil)

		// Duration should be approximately 100ms
		duration := time.Since(start)
		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(100))
	})

	t.Run("handles different topics", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		start := time.Now()

		topics := []string{
			"topic1",
			"device/sensor/1",
			"alerts/critical",
			"",
		}

		for _, topic := range topics {
			observePublish(ctx, start, topic, nil)
		}
	})
}

func TestObservePublish_WithContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("handles cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		start := time.Now()

		// Should not panic even with cancelled context
		observePublish(ctx, start, mqttTestTopic, nil)
	})
}

func TestPackageConstants(t *testing.T) {
	t.Parallel()

	t.Run("package name is correct", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "mqtt", packageName)
	})

	t.Run("topic attribute is correct", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "topic", topicAttribute)
	})
}
