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

func newGetFunc(t Transport) Get {
	return func(index string, id string, o ...func(*GetRequest)) (*Response, error) {
		var r = GetRequest{Index: index, DocumentID: id}
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

// Get returns a document.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/docs-get.html.
type Get func(index string, id string, o ...func(*GetRequest)) (*Response, error)

// GetRequest configures the Get API request.
type GetRequest struct {
	Index      string
	DocumentID string

	ForceSyntheticSource *bool
	Preference           string
	Realtime             *bool
	Refresh              *bool
	Routing              string
	Source               []string
	SourceExcludes       []string
	SourceIncludes       []string
	StoredFields         []string
	Version              *int
	VersionType          string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r GetRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "get")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len(r.Index) + 1 + len("_doc") + 1 + len(r.DocumentID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString(r.Index)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "index", r.Index)
	}
	path.WriteString("/")
	path.WriteString("_doc")
	path.WriteString("/")
	path.WriteString(r.DocumentID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "id", r.DocumentID)
	}

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

	if r.Version != nil {
		params["version"] = strconv.FormatInt(int64(*r.Version), 10)
	}

	if r.VersionType != "" {
		params["version_type"] = r.VersionType
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
		instrument.BeforeRequest(req, "get")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "get")
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
func (f Get) WithContext(v context.Context) func(*GetRequest) {
	return func(r *GetRequest) {
		r.ctx = v
	}
}

// WithForceSyntheticSource - should this request force synthetic _source? use this to test if the mapping supports synthetic _source and to get a sense of the worst case performance. fetches with this enabled will be slower the enabling synthetic source natively in the index..
func (f Get) WithForceSyntheticSource(v bool) func(*GetRequest) {
	return func(r *GetRequest) {
		r.ForceSyntheticSource = &v
	}
}

// WithPreference - specify the node or shard the operation should be performed on (default: random).
func (f Get) WithPreference(v string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Preference = v
	}
}

// WithRealtime - specify whether to perform the operation in realtime or search mode.
func (f Get) WithRealtime(v bool) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Realtime = &v
	}
}

// WithRefresh - refresh the shard containing the document before performing the operation.
func (f Get) WithRefresh(v bool) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Refresh = &v
	}
}

// WithRouting - specific routing value.
func (f Get) WithRouting(v string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Routing = v
	}
}

// WithSource - true or false to return the _source field or not, or a list of fields to return.
func (f Get) WithSource(v ...string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Source = v
	}
}

// WithSourceExcludes - a list of fields to exclude from the returned _source field.
func (f Get) WithSourceExcludes(v ...string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.SourceExcludes = v
	}
}

// WithSourceIncludes - a list of fields to extract and return from the _source field.
func (f Get) WithSourceIncludes(v ...string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.SourceIncludes = v
	}
}

// WithStoredFields - a list of stored fields to return in the response.
func (f Get) WithStoredFields(v ...string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.StoredFields = v
	}
}

// WithVersion - explicit version number for concurrency control.
func (f Get) WithVersion(v int) func(*GetRequest) {
	return func(r *GetRequest) {
		r.Version = &v
	}
}

// WithVersionType - specific version type.
func (f Get) WithVersionType(v string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.VersionType = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f Get) WithPretty() func(*GetRequest) {
	return func(r *GetRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f Get) WithHuman() func(*GetRequest) {
	return func(r *GetRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f Get) WithErrorTrace() func(*GetRequest) {
	return func(r *GetRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f Get) WithFilterPath(v ...string) func(*GetRequest) {
	return func(r *GetRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f Get) WithHeader(h map[string]string) func(*GetRequest) {
	return func(r *GetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f Get) WithOpaqueID(s string) func(*GetRequest) {
	return func(r *GetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
