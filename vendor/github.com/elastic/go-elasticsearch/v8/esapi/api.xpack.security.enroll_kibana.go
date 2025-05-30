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
	"net/http"
	"strings"
)

func newSecurityEnrollKibanaFunc(t Transport) SecurityEnrollKibana {
	return func(o ...func(*SecurityEnrollKibanaRequest)) (*Response, error) {
		var r = SecurityEnrollKibanaRequest{}
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

// SecurityEnrollKibana - Allows a kibana instance to configure itself to communicate with a secured elasticsearch cluster.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/security-api-kibana-enrollment.html.
type SecurityEnrollKibana func(o ...func(*SecurityEnrollKibanaRequest)) (*Response, error)

// SecurityEnrollKibanaRequest configures the Security Enroll Kibana API request.
type SecurityEnrollKibanaRequest struct {
	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SecurityEnrollKibanaRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "security.enroll_kibana")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + len("/_security/enroll/kibana"))
	path.WriteString("http://")
	path.WriteString("/_security/enroll/kibana")

	params = make(map[string]string)

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

	req, err := newRequest(method, path.String(), nil)
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

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.BeforeRequest(req, "security.enroll_kibana")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "security.enroll_kibana")
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
func (f SecurityEnrollKibana) WithContext(v context.Context) func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SecurityEnrollKibana) WithPretty() func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SecurityEnrollKibana) WithHuman() func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SecurityEnrollKibana) WithErrorTrace() func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SecurityEnrollKibana) WithFilterPath(v ...string) func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SecurityEnrollKibana) WithHeader(h map[string]string) func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SecurityEnrollKibana) WithOpaqueID(s string) func(*SecurityEnrollKibanaRequest) {
	return func(r *SecurityEnrollKibanaRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
