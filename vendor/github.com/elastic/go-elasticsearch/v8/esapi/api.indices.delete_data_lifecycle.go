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
	"errors"
	"net/http"
	"strings"
	"time"
)

func newIndicesDeleteDataLifecycleFunc(t Transport) IndicesDeleteDataLifecycle {
	return func(name []string, o ...func(*IndicesDeleteDataLifecycleRequest)) (*Response, error) {
		var r = IndicesDeleteDataLifecycleRequest{Name: name}
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

// IndicesDeleteDataLifecycle deletes the data stream lifecycle of the selected data streams.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/data-streams-delete-lifecycle.html.
type IndicesDeleteDataLifecycle func(name []string, o ...func(*IndicesDeleteDataLifecycleRequest)) (*Response, error)

// IndicesDeleteDataLifecycleRequest configures the Indices Delete Data Lifecycle API request.
type IndicesDeleteDataLifecycleRequest struct {
	Name []string

	ExpandWildcards string
	MasterTimeout   time.Duration
	Timeout         time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r IndicesDeleteDataLifecycleRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "indices.delete_data_lifecycle")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "DELETE"

	if len(r.Name) == 0 {
		return nil, errors.New("name is required and cannot be nil or empty")
	}

	path.Grow(7 + 1 + len("_data_stream") + 1 + len(strings.Join(r.Name, ",")) + 1 + len("_lifecycle"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_data_stream")
	path.WriteString("/")
	path.WriteString(strings.Join(r.Name, ","))
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "name", strings.Join(r.Name, ","))
	}
	path.WriteString("/")
	path.WriteString("_lifecycle")

	params = make(map[string]string)

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
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
		instrument.BeforeRequest(req, "indices.delete_data_lifecycle")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "indices.delete_data_lifecycle")
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
func (f IndicesDeleteDataLifecycle) WithContext(v context.Context) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.ctx = v
	}
}

// WithExpandWildcards - whether wildcard expressions should get expanded to open or closed indices (default: open).
func (f IndicesDeleteDataLifecycle) WithExpandWildcards(v string) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.ExpandWildcards = v
	}
}

// WithMasterTimeout - specify timeout for connection to master.
func (f IndicesDeleteDataLifecycle) WithMasterTimeout(v time.Duration) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.MasterTimeout = v
	}
}

// WithTimeout - explicit timestamp for the document.
func (f IndicesDeleteDataLifecycle) WithTimeout(v time.Duration) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f IndicesDeleteDataLifecycle) WithPretty() func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f IndicesDeleteDataLifecycle) WithHuman() func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f IndicesDeleteDataLifecycle) WithErrorTrace() func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f IndicesDeleteDataLifecycle) WithFilterPath(v ...string) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f IndicesDeleteDataLifecycle) WithHeader(h map[string]string) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f IndicesDeleteDataLifecycle) WithOpaqueID(s string) func(*IndicesDeleteDataLifecycleRequest) {
	return func(r *IndicesDeleteDataLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
