package mqtt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
