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
)

func newRenderSearchTemplateFunc(t Transport) RenderSearchTemplate {
	return func(o ...func(*RenderSearchTemplateRequest)) (*Response, error) {
		var r = RenderSearchTemplateRequest{}
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

// RenderSearchTemplate allows to use the Mustache language to pre-render a search definition.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/render-search-template-api.html.
type RenderSearchTemplate func(o ...func(*RenderSearchTemplateRequest)) (*Response, error)

// RenderSearchTemplateRequest configures the Render Search Template API request.
type RenderSearchTemplateRequest struct {
	TemplateID string

	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r RenderSearchTemplateRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "render_search_template")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_render") + 1 + len("template") + 1 + len(r.TemplateID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_render")
	path.WriteString("/")
	path.WriteString("template")
	if r.TemplateID != "" {
		path.WriteString("/")
		path.WriteString(r.TemplateID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "id", r.TemplateID)
		}
	}

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
		instrument.BeforeRequest(req, "render_search_template")
		if reader := instrument.RecordRequestBody(ctx, "render_search_template", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "render_search_template")
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
func (f RenderSearchTemplate) WithContext(v context.Context) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.ctx = v
	}
}

// WithBody - The search definition template and its params.
func (f RenderSearchTemplate) WithBody(v io.Reader) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Body = v
	}
}

// WithTemplateID - the ID of the stored search template.
func (f RenderSearchTemplate) WithTemplateID(v string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.TemplateID = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f RenderSearchTemplate) WithPretty() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f RenderSearchTemplate) WithHuman() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f RenderSearchTemplate) WithErrorTrace() func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f RenderSearchTemplate) WithFilterPath(v ...string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f RenderSearchTemplate) WithHeader(h map[string]string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f RenderSearchTemplate) WithOpaqueID(s string) func(*RenderSearchTemplateRequest) {
	return func(r *RenderSearchTemplateRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
