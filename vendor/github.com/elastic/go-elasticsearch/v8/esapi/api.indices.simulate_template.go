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

func newIndicesSimulateTemplateFunc(t Transport) IndicesSimulateTemplate {
	return func(o ...func(*IndicesSimulateTemplateRequest)) (*Response, error) {
		var r = IndicesSimulateTemplateRequest{}
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

// IndicesSimulateTemplate simulate resolving the given template name or body
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/indices-simulate-template.html.
type IndicesSimulateTemplate func(o ...func(*IndicesSimulateTemplateRequest)) (*Response, error)

// IndicesSimulateTemplateRequest configures the Indices Simulate Template API request.
type IndicesSimulateTemplateRequest struct {
	Body io.Reader

	Name string

	Cause           string
	Create          *bool
	IncludeDefaults *bool
	MasterTimeout   time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r IndicesSimulateTemplateRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "indices.simulate_template")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_index_template") + 1 + len("_simulate") + 1 + len(r.Name))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_index_template")
	path.WriteString("/")
	path.WriteString("_simulate")
	if r.Name != "" {
		path.WriteString("/")
		path.WriteString(r.Name)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "name", r.Name)
		}
	}

	params = make(map[string]string)

	if r.Cause != "" {
		params["cause"] = r.Cause
	}

	if r.Create != nil {
		params["create"] = strconv.FormatBool(*r.Create)
	}

	if r.IncludeDefaults != nil {
		params["include_defaults"] = strconv.FormatBool(*r.IncludeDefaults)
	}

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
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
		instrument.BeforeRequest(req, "indices.simulate_template")
		if reader := instrument.RecordRequestBody(ctx, "indices.simulate_template", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "indices.simulate_template")
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
func (f IndicesSimulateTemplate) WithContext(v context.Context) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.ctx = v
	}
}

// WithBody - New index template definition to be simulated, if no index template name is specified.
func (f IndicesSimulateTemplate) WithBody(v io.Reader) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Body = v
	}
}

// WithName - the name of the index template.
func (f IndicesSimulateTemplate) WithName(v string) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Name = v
	}
}

// WithCause - user defined reason for dry-run creating the new template for simulation purposes.
func (f IndicesSimulateTemplate) WithCause(v string) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Cause = v
	}
}

// WithCreate - whether the index template we optionally defined in the body should only be dry-run added if new or can also replace an existing one.
func (f IndicesSimulateTemplate) WithCreate(v bool) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Create = &v
	}
}

// WithIncludeDefaults - return all relevant default configurations for this template simulation (default: false).
func (f IndicesSimulateTemplate) WithIncludeDefaults(v bool) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.IncludeDefaults = &v
	}
}

// WithMasterTimeout - specify timeout for connection to master.
func (f IndicesSimulateTemplate) WithMasterTimeout(v time.Duration) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.MasterTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f IndicesSimulateTemplate) WithPretty() func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f IndicesSimulateTemplate) WithHuman() func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f IndicesSimulateTemplate) WithErrorTrace() func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f IndicesSimulateTemplate) WithFilterPath(v ...string) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f IndicesSimulateTemplate) WithHeader(h map[string]string) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f IndicesSimulateTemplate) WithOpaqueID(s string) func(*IndicesSimulateTemplateRequest) {
	return func(r *IndicesSimulateTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
