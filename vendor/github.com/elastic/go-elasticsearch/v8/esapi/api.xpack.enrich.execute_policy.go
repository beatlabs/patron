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
	"time"
)

func newEnrichExecutePolicyFunc(t Transport) EnrichExecutePolicy {
	return func(name string, o ...func(*EnrichExecutePolicyRequest)) (*Response, error) {
		var r = EnrichExecutePolicyRequest{Name: name}
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

// EnrichExecutePolicy - Creates the enrich index for an existing enrich policy.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/execute-enrich-policy-api.html.
type EnrichExecutePolicy func(name string, o ...func(*EnrichExecutePolicyRequest)) (*Response, error)

// EnrichExecutePolicyRequest configures the Enrich Execute Policy API request.
type EnrichExecutePolicyRequest struct {
	Name string

	MasterTimeout     time.Duration
	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r EnrichExecutePolicyRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "enrich.execute_policy")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + 1 + len("_enrich") + 1 + len("policy") + 1 + len(r.Name) + 1 + len("_execute"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_enrich")
	path.WriteString("/")
	path.WriteString("policy")
	path.WriteString("/")
	path.WriteString(r.Name)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "name", r.Name)
	}
	path.WriteString("/")
	path.WriteString("_execute")

	params = make(map[string]string)

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if r.WaitForCompletion != nil {
		params["wait_for_completion"] = strconv.FormatBool(*r.WaitForCompletion)
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
		instrument.BeforeRequest(req, "enrich.execute_policy")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "enrich.execute_policy")
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
func (f EnrichExecutePolicy) WithContext(v context.Context) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - timeout for processing on master node.
func (f EnrichExecutePolicy) WithMasterTimeout(v time.Duration) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.MasterTimeout = v
	}
}

// WithWaitForCompletion - should the request should block until the execution is complete..
func (f EnrichExecutePolicy) WithWaitForCompletion(v bool) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f EnrichExecutePolicy) WithPretty() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f EnrichExecutePolicy) WithHuman() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f EnrichExecutePolicy) WithErrorTrace() func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f EnrichExecutePolicy) WithFilterPath(v ...string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f EnrichExecutePolicy) WithHeader(h map[string]string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f EnrichExecutePolicy) WithOpaqueID(s string) func(*EnrichExecutePolicyRequest) {
	return func(r *EnrichExecutePolicyRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
