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
	"time"
)

func newSlmExecuteLifecycleFunc(t Transport) SlmExecuteLifecycle {
	return func(policy_id string, o ...func(*SlmExecuteLifecycleRequest)) (*Response, error) {
		var r = SlmExecuteLifecycleRequest{PolicyID: policy_id}
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

// SlmExecuteLifecycle - Immediately creates a snapshot according to the lifecycle policy, without waiting for the scheduled time.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/slm-api-execute-lifecycle.html.
type SlmExecuteLifecycle func(policy_id string, o ...func(*SlmExecuteLifecycleRequest)) (*Response, error)

// SlmExecuteLifecycleRequest configures the Slm Execute Lifecycle API request.
type SlmExecuteLifecycleRequest struct {
	PolicyID string

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
func (r SlmExecuteLifecycleRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "slm.execute_lifecycle")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + 1 + len("_slm") + 1 + len("policy") + 1 + len(r.PolicyID) + 1 + len("_execute"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_slm")
	path.WriteString("/")
	path.WriteString("policy")
	path.WriteString("/")
	path.WriteString(r.PolicyID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "policy_id", r.PolicyID)
	}
	path.WriteString("/")
	path.WriteString("_execute")

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
		instrument.BeforeRequest(req, "slm.execute_lifecycle")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "slm.execute_lifecycle")
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
func (f SlmExecuteLifecycle) WithContext(v context.Context) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
func (f SlmExecuteLifecycle) WithMasterTimeout(v time.Duration) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - explicit operation timeout.
func (f SlmExecuteLifecycle) WithTimeout(v time.Duration) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SlmExecuteLifecycle) WithPretty() func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SlmExecuteLifecycle) WithHuman() func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SlmExecuteLifecycle) WithErrorTrace() func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SlmExecuteLifecycle) WithFilterPath(v ...string) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SlmExecuteLifecycle) WithHeader(h map[string]string) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SlmExecuteLifecycle) WithOpaqueID(s string) func(*SlmExecuteLifecycleRequest) {
	return func(r *SlmExecuteLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
