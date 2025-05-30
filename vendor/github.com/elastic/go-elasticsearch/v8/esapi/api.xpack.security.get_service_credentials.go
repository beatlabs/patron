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

func newSecurityGetServiceCredentialsFunc(t Transport) SecurityGetServiceCredentials {
	return func(namespace string, service string, o ...func(*SecurityGetServiceCredentialsRequest)) (*Response, error) {
		var r = SecurityGetServiceCredentialsRequest{Namespace: namespace, Service: service}
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

// SecurityGetServiceCredentials - Retrieves information of all service credentials for a service account.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-get-service-credentials.html.
type SecurityGetServiceCredentials func(namespace string, service string, o ...func(*SecurityGetServiceCredentialsRequest)) (*Response, error)

// SecurityGetServiceCredentialsRequest configures the Security Get Service Credentials API request.
type SecurityGetServiceCredentialsRequest struct {
	Namespace string
	Service   string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SecurityGetServiceCredentialsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "security.get_service_credentials")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_security") + 1 + len("service") + 1 + len(r.Namespace) + 1 + len(r.Service) + 1 + len("credential"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_security")
	path.WriteString("/")
	path.WriteString("service")
	path.WriteString("/")
	path.WriteString(r.Namespace)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "namespace", r.Namespace)
	}
	path.WriteString("/")
	path.WriteString(r.Service)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "service", r.Service)
	}
	path.WriteString("/")
	path.WriteString("credential")

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
		instrument.BeforeRequest(req, "security.get_service_credentials")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "security.get_service_credentials")
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
func (f SecurityGetServiceCredentials) WithContext(v context.Context) func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SecurityGetServiceCredentials) WithPretty() func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SecurityGetServiceCredentials) WithHuman() func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SecurityGetServiceCredentials) WithErrorTrace() func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SecurityGetServiceCredentials) WithFilterPath(v ...string) func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SecurityGetServiceCredentials) WithHeader(h map[string]string) func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SecurityGetServiceCredentials) WithOpaqueID(s string) func(*SecurityGetServiceCredentialsRequest) {
	return func(r *SecurityGetServiceCredentialsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
