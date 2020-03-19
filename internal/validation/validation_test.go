package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStringSliceEmpty(t *testing.T) {
	tcases := []struct {
		name       string
		values     []string
		wantResult bool
	}{
		{
			name:       "nil slice",
			values:     nil,
			wantResult: true,
		},
		{
			name:       "empty slice",
			values:     []string{},
			wantResult: true,
		},
		{
			name:       "all values are empty",
			values:     []string{"", ""},
			wantResult: true,
		},
		{
			name:       "one of the values is empty",
			values:     []string{"", "value"},
			wantResult: true,
		},
		{
			name:       "one of the values is only-spaces value",
			values:     []string{"     ", "value"},
			wantResult: true,
		},
		{
			name:       "all values are non-empty",
			values:     []string{"value1", "value2"},
			wantResult: false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantResult, IsStringSliceEmpty(tc.values))
		})
	}
}
