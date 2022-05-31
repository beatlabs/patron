package http

import (
	"fmt"
	"net/http"
)

// Error defines an abstract struct that can represent several types of HTTP errors.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type Error struct {
	code    int
	payload interface{}
	headers map[string]string
}

// Error returns the actual message of the error.
func (e *Error) Error() string {
	if e.payload == nil {
		return fmt.Sprintf("HTTP error with code: %d", e.code)
	}
	return fmt.Sprintf("HTTP error with code: %d payload: %v", e.code, e.payload)
}

// WithHeaders adds headers to the error which will be added to the http response.
func (e *Error) WithHeaders(headers map[string]string) *Error {
	e.headers = headers
	return e
}

// NewValidationError creates a new validation error with default payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewValidationError() *Error {
	return &Error{code: http.StatusBadRequest, payload: http.StatusText(http.StatusBadRequest)}
}

// NewValidationErrorWithPayload creates a new validation error with the specified payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewValidationErrorWithPayload(payload interface{}) *Error {
	return &Error{code: http.StatusBadRequest, payload: payload}
}

// NewUnauthorizedError creates a new validation error with default payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewUnauthorizedError() *Error {
	return &Error{code: http.StatusUnauthorized, payload: http.StatusText(http.StatusUnauthorized)}
}

// NewUnauthorizedErrorWithPayload creates a new unauthorized error with the specified payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewUnauthorizedErrorWithPayload(payload interface{}) *Error {
	return &Error{code: http.StatusUnauthorized, payload: payload}
}

// NewForbiddenError creates a new forbidden error with default payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewForbiddenError() *Error {
	return &Error{code: http.StatusForbidden, payload: http.StatusText(http.StatusForbidden)}
}

// NewForbiddenErrorWithPayload creates a new forbidden error with the specified payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewForbiddenErrorWithPayload(payload interface{}) *Error {
	return &Error{code: http.StatusForbidden, payload: payload}
}

// NewNotFoundError creates a new not found error with default payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewNotFoundError() *Error {
	return &Error{code: http.StatusNotFound, payload: http.StatusText(http.StatusNotFound)}
}

// NewNotFoundErrorWithPayload creates a new not found error with the specified payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewNotFoundErrorWithPayload(payload interface{}) *Error {
	return &Error{code: http.StatusNotFound, payload: payload}
}

// NewServiceUnavailableError creates a new service unavailable error with default payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewServiceUnavailableError() *Error {
	return &Error{code: http.StatusServiceUnavailable, payload: http.StatusText(http.StatusServiceUnavailable)}
}

// NewServiceUnavailableErrorWithPayload creates a new service unavailable error with the specified payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewServiceUnavailableErrorWithPayload(payload interface{}) *Error {
	return &Error{code: http.StatusServiceUnavailable, payload: payload}
}

// NewError creates a new error with default Internal Server Error payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewError() *Error {
	return &Error{code: http.StatusInternalServerError, payload: http.StatusText(http.StatusInternalServerError)}
}

// NewErrorWithCodeAndPayload creates a fully customizable error with the specified status code and payload.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewErrorWithCodeAndPayload(code int, payload interface{}) *Error {
	return &Error{code: code, payload: payload}
}
