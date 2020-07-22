// Package errors provides useful error handling implementations.
package errors

import (
	"strings"
)

// Aggregate of errors into one.
type aggregate []error

// Error returns the string representation of the aggregated errors.
func (a aggregate) Error() string {
	b := strings.Builder{}
	for _, err := range a {
		b.WriteString(err.Error())
		b.WriteRune('\n')
	}
	return b.String()
}

// Aggregate errors into one error.
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
