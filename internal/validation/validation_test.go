package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStringSliceEmpty(t *testing.T) {
	tests := map[string]struct {
		values     []string
		wantResult bool
	}{
		"nil slice":                              {values: nil, wantResult: true},
		"empty slice":                            {values: []string{}, wantResult: true},
		"all values are empty":                   {values: []string{"", ""}, wantResult: true},
		"one of the values is empty":             {values: []string{"", "value"}, wantResult: true},
		"one of the values is only-spaces value": {values: []string{"     ", "value"}, wantResult: true},
		"all values are non-empty":               {values: []string{"value1", "value2"}, wantResult: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.wantResult, IsStringSliceEmpty(tt.values))
		})
	}
}
