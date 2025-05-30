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

func newTransformGetTransformFunc(t Transport) TransformGetTransform {
	return func(o ...func(*TransformGetTransformRequest)) (*Response, error) {
		var r = TransformGetTransformRequest{}
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

// TransformGetTransform - Retrieves configuration information for transforms.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-transform.html.
type TransformGetTransform func(o ...func(*TransformGetTransformRequest)) (*Response, error)

// TransformGetTransformRequest configures the Transform Get Transform API request.
type TransformGetTransformRequest struct {
	TransformID string

	AllowNoMatch     *bool
	ExcludeGenerated *bool
	From             *int
	Size             *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r TransformGetTransformRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "transform.get_transform")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_transform") + 1 + len(r.TransformID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_transform")
	if r.TransformID != "" {
		path.WriteString("/")
		path.WriteString(r.TransformID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "transform_id", r.TransformID)
		}
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.ExcludeGenerated != nil {
		params["exclude_generated"] = strconv.FormatBool(*r.ExcludeGenerated)
	}

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
		instrument.BeforeRequest(req, "transform.get_transform")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "transform.get_transform")
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
func (f TransformGetTransform) WithContext(v context.Context) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.ctx = v
	}
}

// WithTransformID - the ID or comma delimited list of ID expressions of the transforms to get, '_all' or '*' implies get all transforms.
func (f TransformGetTransform) WithTransformID(v string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.TransformID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no transforms. (this includes `_all` string or when no transforms have been specified).
func (f TransformGetTransform) WithAllowNoMatch(v bool) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.AllowNoMatch = &v
	}
}

// WithExcludeGenerated - omits fields that are illegal to set on transform put.
func (f TransformGetTransform) WithExcludeGenerated(v bool) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.ExcludeGenerated = &v
	}
}

// WithFrom - skips a number of transform configs, defaults to 0.
func (f TransformGetTransform) WithFrom(v int) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of transforms to get, defaults to 100.
func (f TransformGetTransform) WithSize(v int) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f TransformGetTransform) WithPretty() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f TransformGetTransform) WithHuman() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f TransformGetTransform) WithErrorTrace() func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f TransformGetTransform) WithFilterPath(v ...string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f TransformGetTransform) WithHeader(h map[string]string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f TransformGetTransform) WithOpaqueID(s string) func(*TransformGetTransformRequest) {
	return func(r *TransformGetTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
