package apikey

import (
	"errors"
	"net/http"
	"strings"
)

// Validator interface for validating keys.
type Validator interface {
	Validate(key string) (bool, error)
}

// Authenticator authenticates the request based on the header on the following header key and value:
// Authorization: Apikey {api key}, where {api key} is the key.
type Authenticator struct {
	val Validator
}

// New constructor.
func New(val Validator) (*Authenticator, error) {
	if val == nil {
		return nil, errors.New("validator is nil")
	}
	return &Authenticator{val: val}, nil
}

// Authenticate parses the header for the specified key and authenticates it.
func (a *Authenticator) Authenticate(req *http.Request) (bool, error) {
	headerVal := req.Header.Get("Authorization")
	if headerVal == "" {
		return false, nil
	}

	auth := strings.SplitN(headerVal, " ", 2)
	if len(auth) != 2 {
		return false, nil
	}

	if strings.ToLower(auth[0]) != "apikey" {
		return false, nil
	}

	return a.val.Validate(auth[1])
}
