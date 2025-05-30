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

func newWatcherStatsFunc(t Transport) WatcherStats {
	return func(o ...func(*WatcherStatsRequest)) (*Response, error) {
		var r = WatcherStatsRequest{}
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

// WatcherStats - Retrieves the current Watcher metrics.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/watcher-api-stats.html.
type WatcherStats func(o ...func(*WatcherStatsRequest)) (*Response, error)

// WatcherStatsRequest configures the Watcher Stats API request.
type WatcherStatsRequest struct {
	Metric []string

	EmitStacktraces *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r WatcherStatsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "watcher.stats")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_watcher") + 1 + len("stats") + 1 + len(strings.Join(r.Metric, ",")))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_watcher")
	path.WriteString("/")
	path.WriteString("stats")
	if len(r.Metric) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Metric, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "metric", strings.Join(r.Metric, ","))
		}
	}

	params = make(map[string]string)

	if r.EmitStacktraces != nil {
		params["emit_stacktraces"] = strconv.FormatBool(*r.EmitStacktraces)
	}

	if len(r.Metric) > 0 {
		params["metric"] = strings.Join(r.Metric, ",")
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
		instrument.BeforeRequest(req, "watcher.stats")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "watcher.stats")
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
func (f WatcherStats) WithContext(v context.Context) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.ctx = v
	}
}

// WithMetric - controls what additional stat metrics should be include in the response.
func (f WatcherStats) WithMetric(v ...string) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.Metric = v
	}
}

// WithEmitStacktraces - emits stack traces of currently running watches.
func (f WatcherStats) WithEmitStacktraces(v bool) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.EmitStacktraces = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f WatcherStats) WithPretty() func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f WatcherStats) WithHuman() func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f WatcherStats) WithErrorTrace() func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f WatcherStats) WithFilterPath(v ...string) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f WatcherStats) WithHeader(h map[string]string) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f WatcherStats) WithOpaqueID(s string) func(*WatcherStatsRequest) {
	return func(r *WatcherStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
