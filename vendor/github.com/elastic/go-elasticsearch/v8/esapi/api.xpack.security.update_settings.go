// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// Code generated from specification version 8.18.0: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

func newSecurityUpdateSettingsFunc(t Transport) SecurityUpdateSettings {
	return func(body io.Reader, o ...func(*SecurityUpdateSettingsRequest)) (*Response, error) {
		var r = SecurityUpdateSettingsRequest{Body: body}
		for _, f := range o {
			f(&r)
		}

		if transport, ok := t.(Instrumented); ok {
			r.Instrument = transport.InstrumentationEnabled()
		}

		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SecurityUpdateSettings - Update settings for the security system index
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-update-settings.html.
type SecurityUpdateSettings func(body io.Reader, o ...func(*SecurityUpdateSettingsRequest)) (*Response, error)

// SecurityUpdateSettingsRequest configures the Security Update Settings API request.
type SecurityUpdateSettingsRequest struct {
	Body io.Reader

	MasterTimeout time.Duration
	Timeout       time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SecurityUpdateSettingsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "security.update_settings")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + len("/_security/settings"))
	path.WriteString("http://")
	path.WriteString("/_security/settings")

	params = make(map[string]string)

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.Pretty {
		params["pretty"] = "true"
	}

	if r.Human {
		params["human"] = "true"
	}

	if r.ErrorTrace {
		params["error_trace"] = "true"
	}

	if len(r.FilterPath) > 0 {
		params["filter_path"] = strings.Join(r.FilterPath, ",")
	}

	req, err := newRequest(method, path.String(), r.Body)
	if err != nil {
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordError(ctx, err)
		}
		return nil, err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
	}

	if r.Body != nil && req.Header.Get(headerContentType) == "" {
		req.Header[headerContentType] = headerContentTypeJSON
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.BeforeRequest(req, "security.update_settings")
		if reader := instrument.RecordRequestBody(ctx, "security.update_settings", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "security.update_settings")
	}
	if err != nil {
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordError(ctx, err)
		}
		return nil, err
	}

	response := Response{
		StatusCode: res.StatusCode,
		Body:       res.Body,
		Header:     res.Header,
	}

	return &response, nil
}

// WithContext sets the request context.
func (f SecurityUpdateSettings) WithContext(v context.Context) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - timeout for connection to master.
func (f SecurityUpdateSettings) WithMasterTimeout(v time.Duration) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - timeout for acknowledgements from all nodes.
func (f SecurityUpdateSettings) WithTimeout(v time.Duration) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SecurityUpdateSettings) WithPretty() func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SecurityUpdateSettings) WithHuman() func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SecurityUpdateSettings) WithErrorTrace() func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SecurityUpdateSettings) WithFilterPath(v ...string) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SecurityUpdateSettings) WithHeader(h map[string]string) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SecurityUpdateSettings) WithOpaqueID(s string) func(*SecurityUpdateSettingsRequest) {
	return func(r *SecurityUpdateSettingsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
