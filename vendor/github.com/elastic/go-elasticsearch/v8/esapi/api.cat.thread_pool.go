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

func newCatThreadPoolFunc(t Transport) CatThreadPool {
	return func(o ...func(*CatThreadPoolRequest)) (*Response, error) {
		var r = CatThreadPoolRequest{}
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

// CatThreadPool returns cluster-wide thread pool statistics per node.
// By default the active, queue and rejected statistics are returned for all thread pools.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/cat-thread-pool.html.
type CatThreadPool func(o ...func(*CatThreadPoolRequest)) (*Response, error)

// CatThreadPoolRequest configures the Cat Thread Pool API request.
type CatThreadPoolRequest struct {
	ThreadPoolPatterns []string

	Format        string
	H             []string
	Help          *bool
	Local         *bool
	MasterTimeout time.Duration
	S             []string
	Time          string
	V             *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r CatThreadPoolRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "cat.thread_pool")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_cat") + 1 + len("thread_pool") + 1 + len(strings.Join(r.ThreadPoolPatterns, ",")))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("thread_pool")
	if len(r.ThreadPoolPatterns) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.ThreadPoolPatterns, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "thread_pool_patterns", strings.Join(r.ThreadPoolPatterns, ","))
		}
	}

	params = make(map[string]string)

	if r.Format != "" {
		params["format"] = r.Format
	}

	if len(r.H) > 0 {
		params["h"] = strings.Join(r.H, ",")
	}

	if r.Help != nil {
		params["help"] = strconv.FormatBool(*r.Help)
	}

	if r.Local != nil {
		params["local"] = strconv.FormatBool(*r.Local)
	}

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if len(r.S) > 0 {
		params["s"] = strings.Join(r.S, ",")
	}

	if r.Time != "" {
		params["time"] = r.Time
	}

	if r.V != nil {
		params["v"] = strconv.FormatBool(*r.V)
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
		instrument.BeforeRequest(req, "cat.thread_pool")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "cat.thread_pool")
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
func (f CatThreadPool) WithContext(v context.Context) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.ctx = v
	}
}

// WithThreadPoolPatterns - a list of regular-expressions to filter the thread pools in the output.
func (f CatThreadPool) WithThreadPoolPatterns(v ...string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.ThreadPoolPatterns = v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
func (f CatThreadPool) WithFormat(v string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Format = v
	}
}

// WithH - comma-separated list of column names to display.
func (f CatThreadPool) WithH(v ...string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
func (f CatThreadPool) WithHelp(v bool) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Help = &v
	}
}

// WithLocal - return local information, do not retrieve the state from master node (default: false).
func (f CatThreadPool) WithLocal(v bool) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Local = &v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
func (f CatThreadPool) WithMasterTimeout(v time.Duration) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.MasterTimeout = v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
func (f CatThreadPool) WithS(v ...string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.S = v
	}
}

// WithTime - the unit in which to display time values.
func (f CatThreadPool) WithTime(v string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
func (f CatThreadPool) WithV(v bool) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CatThreadPool) WithPretty() func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CatThreadPool) WithHuman() func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CatThreadPool) WithErrorTrace() func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CatThreadPool) WithFilterPath(v ...string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CatThreadPool) WithHeader(h map[string]string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CatThreadPool) WithOpaqueID(s string) func(*CatThreadPoolRequest) {
	return func(r *CatThreadPoolRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
