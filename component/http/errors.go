package http

import (
	"fmt"
	"net/http"
)

// Error defines an abstract struct that can represent several types of HTTP errors.
type Error struct {
	code    int
	payload interface{}
}

// Error returns the actual message of the error.
func (e *Error) Error() string {
	if e.payload == nil {
		return fmt.Sprintf("HTTP error with code: %d", e.code)
	}
	return fmt.Sprintf("HTTP error with code: %d payload: %v", e.code, e.payload)
}

// NewValidationError creates a new validation error with default payload.
func NewValidationError() *Error {
	return &Error{http.StatusBadRequest, http.StatusText(http.StatusBadRequest)}
}

// NewValidationErrorWithPayload creates a new validation error with the specified payload.
func NewValidationErrorWithPayload(payload interface{}) *Error {
	return &Error{http.StatusBadRequest, payload}
}

// NewUnauthorizedError creates a new validation error with default payload.
func NewUnauthorizedError() *Error {
	return &Error{http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)}
}

// NewUnauthorizedErrorWithPayload creates a new unauthorized error with the specified payload.
func NewUnauthorizedErrorWithPayload(payload interface{}) *Error {
	return &Error{http.StatusUnauthorized, payload}
}

// NewForbiddenError creates a new forbidden error with default payload.
func NewForbiddenError() *Error {
	return &Error{http.StatusForbidden, http.StatusText(http.StatusForbidden)}
}

// NewForbiddenErrorWithPayload creates a new forbidden error with the specified payload.
func NewForbiddenErrorWithPayload(payload interface{}) *Error {
	return &Error{http.StatusForbidden, payload}
}

// NewNotFoundError creates a new not found error with default payload.
func NewNotFoundError() *Error {
	return &Error{http.StatusNotFound, http.StatusText(http.StatusNotFound)}
}

// NewNotFoundErrorWithPayload creates a new not found error with the specified payload.
func NewNotFoundErrorWithPayload(payload interface{}) *Error {
	return &Error{http.StatusNotFound, payload}
}

// NewServiceUnavailableError creates a new service unavailable error with default payload.
func NewServiceUnavailableError() *Error {
	return &Error{http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)}
}

// NewServiceUnavailableErrorWithPayload creates a new service unavailable error with the specified payload.
func NewServiceUnavailableErrorWithPayload(payload interface{}) *Error {
	return &Error{http.StatusServiceUnavailable, payload}
}

// NewError creates a new error with default Internal Server Error payload.
func NewError() *Error {
	return &Error{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)}
}

// NewErrorWithCodeAndPayload creates a fully customizable error with the specified status code and payload.
func NewErrorWithCodeAndPayload(code int, payload interface{}) *Error {
	return &Error{code, payload}
}
