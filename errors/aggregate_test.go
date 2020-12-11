package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregate(t *testing.T) {
	a := Aggregate(errors.New("error 1"), errors.New("error 2"), nil, errors.New("error 3"))
	assert.Len(t, a, 3)
	assert.Equal(t, "error 1\nerror 2\nerror 3\n", a.Error())
}

func TestAggregate_ReturnsNil(t *testing.T) {
	assert.Nil(t, Aggregate(nil, nil, nil))
}
