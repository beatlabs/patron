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
	"strconv"
	"strings"
)

func newSecurityGetUserFunc(t Transport) SecurityGetUser {
	return func(o ...func(*SecurityGetUserRequest)) (*Response, error) {
		var r = SecurityGetUserRequest{}
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

// SecurityGetUser - Retrieves information about users in the native realm and built-in users.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-get-user.html.
type SecurityGetUser func(o ...func(*SecurityGetUserRequest)) (*Response, error)

// SecurityGetUserRequest configures the Security Get User API request.
type SecurityGetUserRequest struct {
	Username []string

	WithProfileUID *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SecurityGetUserRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "security.get_user")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_security") + 1 + len("user") + 1 + len(strings.Join(r.Username, ",")))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_security")
	path.WriteString("/")
	path.WriteString("user")
	if len(r.Username) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Username, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "username", strings.Join(r.Username, ","))
		}
	}

	params = make(map[string]string)

	if r.WithProfileUID != nil {
		params["with_profile_uid"] = strconv.FormatBool(*r.WithProfileUID)
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
		instrument.BeforeRequest(req, "security.get_user")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "security.get_user")
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
func (f SecurityGetUser) WithContext(v context.Context) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.ctx = v
	}
}

// WithUsername - a list of usernames.
func (f SecurityGetUser) WithUsername(v ...string) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.Username = v
	}
}

// WithWithProfileUID - flag to retrieve profile uid (if exists) associated to the user.
func (f SecurityGetUser) WithWithProfileUID(v bool) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.WithProfileUID = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SecurityGetUser) WithPretty() func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SecurityGetUser) WithHuman() func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SecurityGetUser) WithErrorTrace() func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SecurityGetUser) WithFilterPath(v ...string) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SecurityGetUser) WithHeader(h map[string]string) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SecurityGetUser) WithOpaqueID(s string) func(*SecurityGetUserRequest) {
	return func(r *SecurityGetUserRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
