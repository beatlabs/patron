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
)

func newSearchApplicationSearchFunc(t Transport) SearchApplicationSearch {
	return func(name string, o ...func(*SearchApplicationSearchRequest)) (*Response, error) {
		var r = SearchApplicationSearchRequest{Name: name}
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

// SearchApplicationSearch perform a search against a search application
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/search-application-search.html.
type SearchApplicationSearch func(name string, o ...func(*SearchApplicationSearchRequest)) (*Response, error)

// SearchApplicationSearchRequest configures the Search Application Search API request.
type SearchApplicationSearchRequest struct {
	Body io.Reader

	Name string

	TypedKeys *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SearchApplicationSearchRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "search_application.search")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_application") + 1 + len("search_application") + 1 + len(r.Name) + 1 + len("_search"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_application")
	path.WriteString("/")
	path.WriteString("search_application")
	path.WriteString("/")
	path.WriteString(r.Name)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "name", r.Name)
	}
	path.WriteString("/")
	path.WriteString("_search")

	params = make(map[string]string)

	if r.TypedKeys != nil {
		params["typed_keys"] = strconv.FormatBool(*r.TypedKeys)
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
		instrument.BeforeRequest(req, "search_application.search")
		if reader := instrument.RecordRequestBody(ctx, "search_application.search", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "search_application.search")
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
func (f SearchApplicationSearch) WithContext(v context.Context) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.ctx = v
	}
}

// WithBody - Search parameters, including template parameters that override defaults.
func (f SearchApplicationSearch) WithBody(v io.Reader) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.Body = v
	}
}

// WithTypedKeys - specify whether aggregation and suggester names should be prefixed by their respective types in the response.
func (f SearchApplicationSearch) WithTypedKeys(v bool) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.TypedKeys = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SearchApplicationSearch) WithPretty() func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SearchApplicationSearch) WithHuman() func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SearchApplicationSearch) WithErrorTrace() func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SearchApplicationSearch) WithFilterPath(v ...string) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SearchApplicationSearch) WithHeader(h map[string]string) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SearchApplicationSearch) WithOpaqueID(s string) func(*SearchApplicationSearchRequest) {
	return func(r *SearchApplicationSearchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
