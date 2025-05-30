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

func newWatcherPutWatchFunc(t Transport) WatcherPutWatch {
	return func(id string, o ...func(*WatcherPutWatchRequest)) (*Response, error) {
		var r = WatcherPutWatchRequest{WatchID: id}
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

// WatcherPutWatch - Creates a new watch, or updates an existing one.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/watcher-api-put-watch.html.
type WatcherPutWatch func(id string, o ...func(*WatcherPutWatchRequest)) (*Response, error)

// WatcherPutWatchRequest configures the Watcher Put Watch API request.
type WatcherPutWatchRequest struct {
	WatchID string

	Body io.Reader

	Active        *bool
	IfPrimaryTerm *int
	IfSeqNo       *int
	Version       *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r WatcherPutWatchRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "watcher.put_watch")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + 1 + len("_watcher") + 1 + len("watch") + 1 + len(r.WatchID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_watcher")
	path.WriteString("/")
	path.WriteString("watch")
	path.WriteString("/")
	path.WriteString(r.WatchID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "id", r.WatchID)
	}

	params = make(map[string]string)

	if r.Active != nil {
		params["active"] = strconv.FormatBool(*r.Active)
	}

	if r.IfPrimaryTerm != nil {
		params["if_primary_term"] = strconv.FormatInt(int64(*r.IfPrimaryTerm), 10)
	}

	if r.IfSeqNo != nil {
		params["if_seq_no"] = strconv.FormatInt(int64(*r.IfSeqNo), 10)
	}

	if r.Version != nil {
		params["version"] = strconv.FormatInt(int64(*r.Version), 10)
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
		instrument.BeforeRequest(req, "watcher.put_watch")
		if reader := instrument.RecordRequestBody(ctx, "watcher.put_watch", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "watcher.put_watch")
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
func (f WatcherPutWatch) WithContext(v context.Context) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.ctx = v
	}
}

// WithBody - The watch.
func (f WatcherPutWatch) WithBody(v io.Reader) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.Body = v
	}
}

// WithActive - specify whether the watch is in/active by default.
func (f WatcherPutWatch) WithActive(v bool) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.Active = &v
	}
}

// WithIfPrimaryTerm - only update the watch if the last operation that has changed the watch has the specified primary term.
func (f WatcherPutWatch) WithIfPrimaryTerm(v int) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.IfPrimaryTerm = &v
	}
}

// WithIfSeqNo - only update the watch if the last operation that has changed the watch has the specified sequence number.
func (f WatcherPutWatch) WithIfSeqNo(v int) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.IfSeqNo = &v
	}
}

// WithVersion - explicit version number for concurrency control.
func (f WatcherPutWatch) WithVersion(v int) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.Version = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f WatcherPutWatch) WithPretty() func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f WatcherPutWatch) WithHuman() func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f WatcherPutWatch) WithErrorTrace() func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f WatcherPutWatch) WithFilterPath(v ...string) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f WatcherPutWatch) WithHeader(h map[string]string) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f WatcherPutWatch) WithOpaqueID(s string) func(*WatcherPutWatchRequest) {
	return func(r *WatcherPutWatchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
