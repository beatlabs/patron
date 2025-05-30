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

func newIndicesForcemergeFunc(t Transport) IndicesForcemerge {
	return func(o ...func(*IndicesForcemergeRequest)) (*Response, error) {
		var r = IndicesForcemergeRequest{}
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

// IndicesForcemerge performs the force merge operation on one or more indices.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/indices-forcemerge.html.
type IndicesForcemerge func(o ...func(*IndicesForcemergeRequest)) (*Response, error)

// IndicesForcemergeRequest configures the Indices Forcemerge API request.
type IndicesForcemergeRequest struct {
	Index []string

	AllowNoIndices     *bool
	ExpandWildcards    string
	Flush              *bool
	IgnoreUnavailable  *bool
	MaxNumSegments     *int
	OnlyExpungeDeletes *bool
	WaitForCompletion  *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r IndicesForcemergeRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "indices.forcemerge")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len(strings.Join(r.Index, ",")) + 1 + len("_forcemerge"))
	path.WriteString("http://")
	if len(r.Index) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Index, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "index", strings.Join(r.Index, ","))
		}
	}
	path.WriteString("/")
	path.WriteString("_forcemerge")

	params = make(map[string]string)

	if r.AllowNoIndices != nil {
		params["allow_no_indices"] = strconv.FormatBool(*r.AllowNoIndices)
	}

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.Flush != nil {
		params["flush"] = strconv.FormatBool(*r.Flush)
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
	}

	if r.MaxNumSegments != nil {
		params["max_num_segments"] = strconv.FormatInt(int64(*r.MaxNumSegments), 10)
	}

	if r.OnlyExpungeDeletes != nil {
		params["only_expunge_deletes"] = strconv.FormatBool(*r.OnlyExpungeDeletes)
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
		instrument.BeforeRequest(req, "indices.forcemerge")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "indices.forcemerge")
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
func (f IndicesForcemerge) WithContext(v context.Context) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.ctx = v
	}
}

// WithIndex - a list of index names; use _all to perform the operation on all indices.
func (f IndicesForcemerge) WithIndex(v ...string) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.Index = v
	}
}

// WithAllowNoIndices - whether to ignore if a wildcard indices expression resolves into no concrete indices. (this includes `_all` string or when no indices have been specified).
func (f IndicesForcemerge) WithAllowNoIndices(v bool) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.AllowNoIndices = &v
	}
}

// WithExpandWildcards - whether to expand wildcard expression to concrete indices that are open, closed or both..
func (f IndicesForcemerge) WithExpandWildcards(v string) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.ExpandWildcards = v
	}
}

// WithFlush - specify whether the index should be flushed after performing the operation (default: true).
func (f IndicesForcemerge) WithFlush(v bool) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.Flush = &v
	}
}

// WithIgnoreUnavailable - whether specified concrete indices should be ignored when unavailable (missing or closed).
func (f IndicesForcemerge) WithIgnoreUnavailable(v bool) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithMaxNumSegments - the number of segments the index should be merged into (default: dynamic).
func (f IndicesForcemerge) WithMaxNumSegments(v int) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.MaxNumSegments = &v
	}
}

// WithOnlyExpungeDeletes - specify whether the operation should only expunge deleted documents.
func (f IndicesForcemerge) WithOnlyExpungeDeletes(v bool) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.OnlyExpungeDeletes = &v
	}
}

// WithWaitForCompletion - should the request wait until the force merge is completed..
func (f IndicesForcemerge) WithWaitForCompletion(v bool) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f IndicesForcemerge) WithPretty() func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f IndicesForcemerge) WithHuman() func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f IndicesForcemerge) WithErrorTrace() func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f IndicesForcemerge) WithFilterPath(v ...string) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f IndicesForcemerge) WithHeader(h map[string]string) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f IndicesForcemerge) WithOpaqueID(s string) func(*IndicesForcemergeRequest) {
	return func(r *IndicesForcemergeRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
