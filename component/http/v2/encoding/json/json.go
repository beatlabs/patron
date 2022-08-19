// Package json contains helper methods to handler requests and responses more easily.
package json

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/log"
)

const (
	singleStar = "*"
	doubleStar = "*/*"
)

// ReadRequest validates the request headers and decodes into the provided payload.
func ReadRequest(req *http.Request, payload interface{}) error {
	err := validatingJsonContentType(req)
	if err != nil {
		return err
	}

	err = json.Decode(req.Body, payload)
	if err != nil {
		return err
	}
	defer func() {
		err := req.Body.Close()
		if err != nil {
			log.FromContext(req.Context()).Errorf("failed to close request body: %v", err)
		}
	}()

	return nil
}

func validatingJsonContentType(req *http.Request) error {
	header, ok := req.Header[encoding.ContentTypeHeader]
	if !ok {
		return nil
	}

	if len(header) == 0 {
		return nil
	}

	switch header[0] {
	case singleStar, doubleStar, json.Type, json.TypeCharset:
		return nil
	default:
		return fmt.Errorf("invalid content type provided: %s", header[0])
	}
}

// WriteResponse validates the request headers and encodes the provided payload into the response.
func WriteResponse(req *http.Request, w http.ResponseWriter, status int, payload interface{}) error {
	err := validatingJsonAccept(req)
	if err != nil {
		return err
	}

	w.WriteHeader(status)
	w.Header().Add(encoding.ContentTypeHeader, json.Type)

	buf, err := json.Encode(payload)
	if err != nil {
		return err
	}

	_, err = w.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write to response: %w", err)
	}

	return nil
}

func validatingJsonAccept(req *http.Request) error {
	header, ok := req.Header[encoding.AcceptHeader]
	if !ok {
		return nil
	}

	if len(header) == 0 {
		return nil
	}

	if header[0] == "" {
		return nil
	}

	jsonHeaderFound := false

	ah := getMultiValueHeaders(header[0])
	for _, v := range ah {
		if isValidAcceptHeader(v) {
			jsonHeaderFound = true
			break
		}
	}

	if !jsonHeaderFound {
		return fmt.Errorf("invalid accept header: %s", header[0])
	}

	return nil
}

func isValidAcceptHeader(header string) bool {
	parts := strings.SplitN(header, ";", 2)
	switch parts[0] {
	case singleStar, doubleStar, "identity", json.Type, json.TypeCharset:
		return true
	default:
		return false
	}
}

func getMultiValueHeaders(header string) []string {
	if !strings.Contains(header, ",") {
		return []string{header}
	}

	splitHeaders := strings.Split(header, ",")

	trimmedHeaders := make([]string, 0, len(splitHeaders))
	for _, v := range splitHeaders {
		trimmedHeaders = append(trimmedHeaders, strings.TrimSpace(v))
	}

	return trimmedHeaders
}
