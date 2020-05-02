package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	err := NewValidationError()
	assert.EqualError(t, err, "HTTP error with code: 400 Payload: Bad Request")
	assert.Equal(t, 400, err.code)
}

func TestValidationErrorWithPayload(t *testing.T) {
	err := NewValidationErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 400 Payload: test")
	assert.Equal(t, 400, err.code)
}
func TestUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError()
	assert.EqualError(t, err, "HTTP error with code: 401 Payload: Unauthorized")
	assert.Equal(t, 401, err.code)
}

func TestUnauthorizedErrorWithPayload(t *testing.T) {
	err := NewUnauthorizedErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 401 Payload: test")
	assert.Equal(t, 401, err.code)
}

func TestForbiddenError(t *testing.T) {
	err := NewForbiddenError()
	assert.EqualError(t, err, "HTTP error with code: 403 Payload: Forbidden")
	assert.Equal(t, 403, err.code)
}

func TestForbiddenErrorWithPayload(t *testing.T) {
	err := NewForbiddenErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 403 Payload: test")
	assert.Equal(t, 403, err.code)
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError()
	assert.EqualError(t, err, "HTTP error with code: 404 Payload: Not Found")
	assert.Equal(t, 404, err.code)
}

func TestNotFoundErrorWithPayload(t *testing.T) {
	err := NewNotFoundErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 404 Payload: test")
	assert.Equal(t, 404, err.code)
}

func TestServiceUnavailableError(t *testing.T) {
	err := NewServiceUnavailableError()
	assert.EqualError(t, err, "HTTP error with code: 503 Payload: Service Unavailable")
	assert.Equal(t, 503, err.code)
}

func TestServiceUnavailableErrorWithPayload(t *testing.T) {
	err := NewServiceUnavailableErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 503 Payload: test")
	assert.Equal(t, 503, err.code)
}

func TestNewError(t *testing.T) {
	err := NewError()
	assert.EqualError(t, err, "HTTP error with code: 500 Payload: Internal Server Error")
	assert.Equal(t, 500, err.code)
}

func TestNewErrorWithCodeAndPayload(t *testing.T) {
	err := NewErrorWithCodeAndPayload(409, "Conflict")
	assert.EqualError(t, err, "HTTP error with code: 409 Payload: Conflict")
	assert.Equal(t, 409, err.code)
}

func TestNewErrorWithCodeAndNoPayload(t *testing.T) {
	err := NewErrorWithCodeAndPayload(409, nil)
	assert.EqualError(t, err, "HTTP error with code: 409")
	assert.Equal(t, 409, err.code)
}
