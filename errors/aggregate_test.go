package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregate(t *testing.T) {
	a := Aggregate(errors.New("Error 1"), errors.New("Error 2"), nil, errors.New("Error 3"))
	assert.Len(t, a, 3)
	assert.Equal(t, "Error 1\nError 2\nError 3\n", a.Error())
}

func TestAggregate_ReturnsNil(t *testing.T) {
	assert.Nil(t, Aggregate(nil, nil, nil))
}
