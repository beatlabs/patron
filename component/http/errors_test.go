package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_WithHeaders(t *testing.T) {
	header := "Retry-After"
	err := NewErrorWithCodeAndPayload(http.StatusTooManyRequests, "wait").
		WithHeaders(map[string]string{header: "1628002707"})
	assert.EqualError(t, err, "HTTP error with code: 429 payload: wait")
	assert.Equal(t, 429, err.code)
	assert.Equal(t, "1628002707", err.headers[header])
}

func TestValidationError(t *testing.T) {
	err := NewValidationError()
	assert.EqualError(t, err, "HTTP error with code: 400 payload: Bad Request")
	assert.Equal(t, 400, err.code)
}

func TestValidationErrorWithPayload(t *testing.T) {
	err := NewValidationErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 400 payload: test")
	assert.Equal(t, 400, err.code)
}
func TestUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError()
	assert.EqualError(t, err, "HTTP error with code: 401 payload: Unauthorized")
	assert.Equal(t, 401, err.code)
}

func TestUnauthorizedErrorWithPayload(t *testing.T) {
	err := NewUnauthorizedErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 401 payload: test")
	assert.Equal(t, 401, err.code)
}

func TestForbiddenError(t *testing.T) {
	err := NewForbiddenError()
	assert.EqualError(t, err, "HTTP error with code: 403 payload: Forbidden")
	assert.Equal(t, 403, err.code)
}

func TestForbiddenErrorWithPayload(t *testing.T) {
	err := NewForbiddenErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 403 payload: test")
	assert.Equal(t, 403, err.code)
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError()
	assert.EqualError(t, err, "HTTP error with code: 404 payload: Not Found")
	assert.Equal(t, 404, err.code)
}

func TestNotFoundErrorWithPayload(t *testing.T) {
	err := NewNotFoundErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 404 payload: test")
	assert.Equal(t, 404, err.code)
}

func TestServiceUnavailableError(t *testing.T) {
	err := NewServiceUnavailableError()
	assert.EqualError(t, err, "HTTP error with code: 503 payload: Service Unavailable")
	assert.Equal(t, 503, err.code)
}

func TestServiceUnavailableErrorWithPayload(t *testing.T) {
	err := NewServiceUnavailableErrorWithPayload("test")
	assert.EqualError(t, err, "HTTP error with code: 503 payload: test")
	assert.Equal(t, 503, err.code)
}

func TestNewError(t *testing.T) {
	err := NewError()
	assert.EqualError(t, err, "HTTP error with code: 500 payload: Internal Server Error")
	assert.Equal(t, 500, err.code)
}

func TestNewErrorWithCodeAndPayload(t *testing.T) {
	err := NewErrorWithCodeAndPayload(409, "Conflict")
	assert.EqualError(t, err, "HTTP error with code: 409 payload: Conflict")
	assert.Equal(t, 409, err.code)
}

func TestNewErrorWithCodeAndNoPayload(t *testing.T) {
	err := NewErrorWithCodeAndPayload(409, nil)
	assert.EqualError(t, err, "HTTP error with code: 409")
	assert.Equal(t, 409, err.code)
}
