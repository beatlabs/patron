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

func newMLGetFiltersFunc(t Transport) MLGetFilters {
	return func(o ...func(*MLGetFiltersRequest)) (*Response, error) {
		var r = MLGetFiltersRequest{}
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

// MLGetFilters - Retrieves filters.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-filter.html.
type MLGetFilters func(o ...func(*MLGetFiltersRequest)) (*Response, error)

// MLGetFiltersRequest configures the ML Get Filters API request.
type MLGetFiltersRequest struct {
	FilterID string

	From *int
	Size *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetFiltersRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_filters")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("filters") + 1 + len(r.FilterID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("filters")
	if r.FilterID != "" {
		path.WriteString("/")
		path.WriteString(r.FilterID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "filter_id", r.FilterID)
		}
	}

	params = make(map[string]string)

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
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
		instrument.BeforeRequest(req, "ml.get_filters")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_filters")
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
func (f MLGetFilters) WithContext(v context.Context) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.ctx = v
	}
}

// WithFilterID - the ID of the filter to fetch.
func (f MLGetFilters) WithFilterID(v string) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.FilterID = v
	}
}

// WithFrom - skips a number of filters.
func (f MLGetFilters) WithFrom(v int) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of filters to get.
func (f MLGetFilters) WithSize(v int) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetFilters) WithPretty() func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetFilters) WithHuman() func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetFilters) WithErrorTrace() func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetFilters) WithFilterPath(v ...string) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetFilters) WithHeader(h map[string]string) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetFilters) WithOpaqueID(s string) func(*MLGetFiltersRequest) {
	return func(r *MLGetFiltersRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
