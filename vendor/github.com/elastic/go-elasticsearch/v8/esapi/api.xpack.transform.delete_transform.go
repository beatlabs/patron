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

func newTransformDeleteTransformFunc(t Transport) TransformDeleteTransform {
	return func(transform_id string, o ...func(*TransformDeleteTransformRequest)) (*Response, error) {
		var r = TransformDeleteTransformRequest{TransformID: transform_id}
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

// TransformDeleteTransform - Deletes an existing transform.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-transform.html.
type TransformDeleteTransform func(transform_id string, o ...func(*TransformDeleteTransformRequest)) (*Response, error)

// TransformDeleteTransformRequest configures the Transform Delete Transform API request.
type TransformDeleteTransformRequest struct {
	TransformID string

	DeleteDestIndex *bool
	Force           *bool
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
func (r TransformDeleteTransformRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "transform.delete_transform")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "DELETE"

	path.Grow(7 + 1 + len("_transform") + 1 + len(r.TransformID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_transform")
	path.WriteString("/")
	path.WriteString(r.TransformID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "transform_id", r.TransformID)
	}

	params = make(map[string]string)

	if r.DeleteDestIndex != nil {
		params["delete_dest_index"] = strconv.FormatBool(*r.DeleteDestIndex)
	}

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
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
		instrument.BeforeRequest(req, "transform.delete_transform")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "transform.delete_transform")
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
func (f TransformDeleteTransform) WithContext(v context.Context) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.ctx = v
	}
}

// WithDeleteDestIndex - when `true`, the destination index is deleted together with the transform. the default value is `false`, meaning that the destination index will not be deleted..
func (f TransformDeleteTransform) WithDeleteDestIndex(v bool) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.DeleteDestIndex = &v
	}
}

// WithForce - when `true`, the transform is deleted regardless of its current state. the default value is `false`, meaning that the transform must be `stopped` before it can be deleted..
func (f TransformDeleteTransform) WithForce(v bool) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.Force = &v
	}
}

// WithTimeout - controls the time to wait for the transform deletion.
func (f TransformDeleteTransform) WithTimeout(v time.Duration) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f TransformDeleteTransform) WithPretty() func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f TransformDeleteTransform) WithHuman() func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f TransformDeleteTransform) WithErrorTrace() func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f TransformDeleteTransform) WithFilterPath(v ...string) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f TransformDeleteTransform) WithHeader(h map[string]string) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f TransformDeleteTransform) WithOpaqueID(s string) func(*TransformDeleteTransformRequest) {
	return func(r *TransformDeleteTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
