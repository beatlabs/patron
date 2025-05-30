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
	"strconv"
	"strings"
	"time"
)

func newLicensePostFunc(t Transport) LicensePost {
	return func(o ...func(*LicensePostRequest)) (*Response, error) {
		var r = LicensePostRequest{}
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

// LicensePost - Updates the license for the cluster.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/update-license.html.
type LicensePost func(o ...func(*LicensePostRequest)) (*Response, error)

// LicensePostRequest configures the License Post API request.
type LicensePostRequest struct {
	Body io.Reader

	Acknowledge   *bool
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
func (r LicensePostRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "license.post")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + len("/_license"))
	path.WriteString("http://")
	path.WriteString("/_license")

	params = make(map[string]string)

	if r.Acknowledge != nil {
		params["acknowledge"] = strconv.FormatBool(*r.Acknowledge)
	}

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
		instrument.BeforeRequest(req, "license.post")
		if reader := instrument.RecordRequestBody(ctx, "license.post", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "license.post")
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
func (f LicensePost) WithContext(v context.Context) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.ctx = v
	}
}

// WithBody - licenses to be installed.
func (f LicensePost) WithBody(v io.Reader) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.Body = v
	}
}

// WithAcknowledge - whether the user has acknowledged acknowledge messages (default: false).
func (f LicensePost) WithAcknowledge(v bool) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.Acknowledge = &v
	}
}

// WithMasterTimeout - timeout for processing on master node.
func (f LicensePost) WithMasterTimeout(v time.Duration) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - timeout for acknowledgement of update from all nodes in cluster.
func (f LicensePost) WithTimeout(v time.Duration) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f LicensePost) WithPretty() func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f LicensePost) WithHuman() func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f LicensePost) WithErrorTrace() func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f LicensePost) WithFilterPath(v ...string) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f LicensePost) WithHeader(h map[string]string) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f LicensePost) WithOpaqueID(s string) func(*LicensePostRequest) {
	return func(r *LicensePostRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
