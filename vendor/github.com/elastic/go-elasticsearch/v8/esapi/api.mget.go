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

func newMgetFunc(t Transport) Mget {
	return func(body io.Reader, o ...func(*MgetRequest)) (*Response, error) {
		var r = MgetRequest{Body: body}
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

// Mget allows to get multiple documents in one request.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/docs-multi-get.html.
type Mget func(body io.Reader, o ...func(*MgetRequest)) (*Response, error)

// MgetRequest configures the Mget API request.
type MgetRequest struct {
	Index string

	Body io.Reader

	ForceSyntheticSource *bool
	Preference           string
	Realtime             *bool
	Refresh              *bool
	Routing              string
	Source               []string
	SourceExcludes       []string
	SourceIncludes       []string
	StoredFields         []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MgetRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "mget")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len(r.Index) + 1 + len("_mget"))
	path.WriteString("http://")
	if r.Index != "" {
		path.WriteString("/")
		path.WriteString(r.Index)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "index", r.Index)
		}
	}
	path.WriteString("/")
	path.WriteString("_mget")

	params = make(map[string]string)

	if r.ForceSyntheticSource != nil {
		params["force_synthetic_source"] = strconv.FormatBool(*r.ForceSyntheticSource)
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.Realtime != nil {
		params["realtime"] = strconv.FormatBool(*r.Realtime)
	}

	if r.Refresh != nil {
		params["refresh"] = strconv.FormatBool(*r.Refresh)
	}

	if r.Routing != "" {
		params["routing"] = r.Routing
	}

	if len(r.Source) > 0 {
		params["_source"] = strings.Join(r.Source, ",")
	}

	if len(r.SourceExcludes) > 0 {
		params["_source_excludes"] = strings.Join(r.SourceExcludes, ",")
	}

	if len(r.SourceIncludes) > 0 {
		params["_source_includes"] = strings.Join(r.SourceIncludes, ",")
	}

	if len(r.StoredFields) > 0 {
		params["stored_fields"] = strings.Join(r.StoredFields, ",")
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
		instrument.BeforeRequest(req, "mget")
		if reader := instrument.RecordRequestBody(ctx, "mget", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "mget")
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
func (f Mget) WithContext(v context.Context) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.ctx = v
	}
}

// WithIndex - the name of the index.
func (f Mget) WithIndex(v string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Index = v
	}
}

// WithForceSyntheticSource - should this request force synthetic _source? use this to test if the mapping supports synthetic _source and to get a sense of the worst case performance. fetches with this enabled will be slower the enabling synthetic source natively in the index..
func (f Mget) WithForceSyntheticSource(v bool) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.ForceSyntheticSource = &v
	}
}

// WithPreference - specify the node or shard the operation should be performed on (default: random).
func (f Mget) WithPreference(v string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Preference = v
	}
}

// WithRealtime - specify whether to perform the operation in realtime or search mode.
func (f Mget) WithRealtime(v bool) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Realtime = &v
	}
}

// WithRefresh - refresh the shard containing the document before performing the operation.
func (f Mget) WithRefresh(v bool) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Refresh = &v
	}
}

// WithRouting - specific routing value.
func (f Mget) WithRouting(v string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Routing = v
	}
}

// WithSource - true or false to return the _source field or not, or a list of fields to return.
func (f Mget) WithSource(v ...string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Source = v
	}
}

// WithSourceExcludes - a list of fields to exclude from the returned _source field.
func (f Mget) WithSourceExcludes(v ...string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.SourceExcludes = v
	}
}

// WithSourceIncludes - a list of fields to extract and return from the _source field.
func (f Mget) WithSourceIncludes(v ...string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.SourceIncludes = v
	}
}

// WithStoredFields - a list of stored fields to return in the response.
func (f Mget) WithStoredFields(v ...string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.StoredFields = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f Mget) WithPretty() func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f Mget) WithHuman() func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f Mget) WithErrorTrace() func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f Mget) WithFilterPath(v ...string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f Mget) WithHeader(h map[string]string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f Mget) WithOpaqueID(s string) func(*MgetRequest) {
	return func(r *MgetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
