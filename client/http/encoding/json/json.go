// Package json provides helper functions to handle requests and responses.
package json

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/log"
)

// NewRequest creates a request, encodes the body, and sets the appropriate headers.
func NewRequest(ctx context.Context, method string, url string, payload interface{}) (*http.Request, error) {
	buf, err := json.Encode(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set(encoding.ContentTypeHeader, json.Type)
	req.Header.Set(encoding.ContentLengthHeader, strconv.FormatInt(int64(len(buf)), 10))
	req.Header.Set(encoding.AcceptHeader, json.Type)

	return req, nil
}

// FromResponse checks for valid headers and decodes the payload.
func FromResponse(ctx context.Context, rsp *http.Response, payload interface{}) error {
	err := validateContentTypeHeader(rsp)
	if err != nil {
		return err
	}

	buf, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	defer func() {
		err := rsp.Body.Close()
		if err != nil {
			log.FromContext(ctx).Errorf("failed to close response body: %v", err)
		}
	}()

	err = validateContentLengthHeader(rsp, len(buf))
	if err != nil {
		log.FromContext(ctx).Warn(err)
	}

	return json.DecodeRaw(buf, payload)
}

func validateContentTypeHeader(rsp *http.Response) error {
	header, ok := rsp.Header[encoding.ContentTypeHeader]
	if !ok {
		return errors.New("response content type header key is missing")
	}

	if len(header) == 0 {
		return errors.New("response content type header value is missing")
	}

	switch header[0] {
	case json.Type, json.TypeCharset:
		return nil
	default:
		return fmt.Errorf("invalid content type provided: %s", header[0])
	}
}

func validateContentLengthHeader(rsp *http.Response, length int) error {
	header, ok := rsp.Header[encoding.ContentLengthHeader]
	if !ok {
		return nil
	}

	if len(header) == 0 {
		return nil
	}

	expected, err := strconv.Atoi(header[0])
	if err != nil {
		return fmt.Errorf("failed to convert content length to int: %w", err)
	}

	if expected != length {
		return fmt.Errorf("expected content length is %d but got %d", expected, length)
	}

	return nil
}
