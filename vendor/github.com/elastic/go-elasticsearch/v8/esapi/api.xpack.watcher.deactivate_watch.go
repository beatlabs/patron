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
	"strings"
)

func newWatcherDeactivateWatchFunc(t Transport) WatcherDeactivateWatch {
	return func(watch_id string, o ...func(*WatcherDeactivateWatchRequest)) (*Response, error) {
		var r = WatcherDeactivateWatchRequest{WatchID: watch_id}
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

// WatcherDeactivateWatch - Deactivates a currently active watch.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/watcher-api-deactivate-watch.html.
type WatcherDeactivateWatch func(watch_id string, o ...func(*WatcherDeactivateWatchRequest)) (*Response, error)

// WatcherDeactivateWatchRequest configures the Watcher Deactivate Watch API request.
type WatcherDeactivateWatchRequest struct {
	WatchID string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r WatcherDeactivateWatchRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "watcher.deactivate_watch")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "PUT"

	path.Grow(7 + 1 + len("_watcher") + 1 + len("watch") + 1 + len(r.WatchID) + 1 + len("_deactivate"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_watcher")
	path.WriteString("/")
	path.WriteString("watch")
	path.WriteString("/")
	path.WriteString(r.WatchID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "watch_id", r.WatchID)
	}
	path.WriteString("/")
	path.WriteString("_deactivate")

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
		instrument.BeforeRequest(req, "watcher.deactivate_watch")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "watcher.deactivate_watch")
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
func (f WatcherDeactivateWatch) WithContext(v context.Context) func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f WatcherDeactivateWatch) WithPretty() func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f WatcherDeactivateWatch) WithHuman() func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f WatcherDeactivateWatch) WithErrorTrace() func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f WatcherDeactivateWatch) WithFilterPath(v ...string) func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f WatcherDeactivateWatch) WithHeader(h map[string]string) func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f WatcherDeactivateWatch) WithOpaqueID(s string) func(*WatcherDeactivateWatchRequest) {
	return func(r *WatcherDeactivateWatchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
