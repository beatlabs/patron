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

func newIndicesPutIndexTemplateFunc(t Transport) IndicesPutIndexTemplate {
	return func(name string, body io.Reader, o ...func(*IndicesPutIndexTemplateRequest)) (*Response, error) {
		var r = IndicesPutIndexTemplateRequest{Name: name, Body: body}
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

// IndicesPutIndexTemplate creates or updates an index template.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/indices-put-template.html.
type IndicesPutIndexTemplate func(name string, body io.Reader, o ...func(*IndicesPutIndexTemplateRequest)) (*Response, error)

// IndicesPutIndexTemplateRequest configures the Indices Put Index Template API request.
type IndicesPutIndexTemplateRequest struct {
	Body io.Reader

	Name string

	Cause         string
	Create        *bool
	MasterTimeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r IndicesPutIndexTemplateRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "indices.put_index_template")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + 1 + len("_index_template") + 1 + len(r.Name))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_index_template")
	path.WriteString("/")
	path.WriteString(r.Name)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "name", r.Name)
	}

	params = make(map[string]string)

	if r.Cause != "" {
		params["cause"] = r.Cause
	}

	if r.Create != nil {
		params["create"] = strconv.FormatBool(*r.Create)
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
		instrument.BeforeRequest(req, "indices.put_index_template")
		if reader := instrument.RecordRequestBody(ctx, "indices.put_index_template", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "indices.put_index_template")
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
func (f IndicesPutIndexTemplate) WithContext(v context.Context) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.ctx = v
	}
}

// WithCause - user defined reason for creating/updating the index template.
func (f IndicesPutIndexTemplate) WithCause(v string) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.Cause = v
	}
}

// WithCreate - whether the index template should only be added if new or can also replace an existing one.
func (f IndicesPutIndexTemplate) WithCreate(v bool) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.Create = &v
	}
}

// WithMasterTimeout - specify timeout for connection to master.
func (f IndicesPutIndexTemplate) WithMasterTimeout(v time.Duration) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.MasterTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f IndicesPutIndexTemplate) WithPretty() func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f IndicesPutIndexTemplate) WithHuman() func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f IndicesPutIndexTemplate) WithErrorTrace() func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f IndicesPutIndexTemplate) WithFilterPath(v ...string) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f IndicesPutIndexTemplate) WithHeader(h map[string]string) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f IndicesPutIndexTemplate) WithOpaqueID(s string) func(*IndicesPutIndexTemplateRequest) {
	return func(r *IndicesPutIndexTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
