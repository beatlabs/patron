// Package errors provides useful error handling implementations.
package errors

import (
	"strings"
)

type aggregate []error

// Error returns the string representation of the aggregated errors.
// Internally it uses a string builder in order to efficiently merge errors into one.
func (a aggregate) Error() string {
	b := strings.Builder{}
	for _, err := range a {
		b.WriteString(err.Error())
		b.WriteRune('\n')
	}
	return b.String()
}

// Aggregate errors into one error.
// If the provided errors contain a nil it will be skipped.
func Aggregate(ee ...error) error {
	agr := make(aggregate, 0, len(ee))
	for _, e := range ee {
		if e == nil {
			continue
		}
		agr = append(agr, e)
	}
	if len(agr) == 0 {
		return nil
	}
	return agr
}
