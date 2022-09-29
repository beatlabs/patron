package errors

import (
	"errors"
	"fmt"
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

var err error

func BenchmarkAggregate(b *testing.B) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := fmt.Errorf("error inner: %w", errors.New("error 3"))

	var innerErr error

	for n := 0; n < b.N; n++ {
		innerErr = Aggregate(err1, nil, err2, nil, err3, nil)
	}
	err = innerErr
}

func ExampleAggregate() {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := fmt.Errorf("error inner: %w", errors.New("error 3"))

	err := Aggregate(err1, nil, err2, nil, err3)

	fmt.Println(err)
	// Output: error 1
	// error 2
	// error inner: error 3
}
