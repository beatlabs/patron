// Package validation provides validation implementations.
package validation

import "strings"

// IsStringSliceEmpty validates a slice of strings for emptiness.
// It returns true either if the slice is empty or when one of the values is empty ( including spaces ).
func IsStringSliceEmpty(values []string) bool {
	if len(values) == 0 {
		return true
	}

	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}

	return false
}
