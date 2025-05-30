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

func newCatTasksFunc(t Transport) CatTasks {
	return func(o ...func(*CatTasksRequest)) (*Response, error) {
		var r = CatTasksRequest{}
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

// CatTasks returns information about the tasks currently executing on one or more nodes in the cluster.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/tasks.html.
type CatTasks func(o ...func(*CatTasksRequest)) (*Response, error)

// CatTasksRequest configures the Cat Tasks API request.
type CatTasksRequest struct {
	Actions           []string
	Detailed          *bool
	Format            string
	H                 []string
	Help              *bool
	Nodes             []string
	ParentTaskID      string
	S                 []string
	Time              string
	Timeout           time.Duration
	V                 *bool
	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r CatTasksRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "cat.tasks")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + len("/_cat/tasks"))
	path.WriteString("http://")
	path.WriteString("/_cat/tasks")

	params = make(map[string]string)

	if len(r.Actions) > 0 {
		params["actions"] = strings.Join(r.Actions, ",")
	}

	if r.Detailed != nil {
		params["detailed"] = strconv.FormatBool(*r.Detailed)
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if len(r.H) > 0 {
		params["h"] = strings.Join(r.H, ",")
	}

	if r.Help != nil {
		params["help"] = strconv.FormatBool(*r.Help)
	}

	if len(r.Nodes) > 0 {
		params["nodes"] = strings.Join(r.Nodes, ",")
	}

	if r.ParentTaskID != "" {
		params["parent_task_id"] = r.ParentTaskID
	}

	if len(r.S) > 0 {
		params["s"] = strings.Join(r.S, ",")
	}

	if r.Time != "" {
		params["time"] = r.Time
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.V != nil {
		params["v"] = strconv.FormatBool(*r.V)
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
		instrument.BeforeRequest(req, "cat.tasks")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "cat.tasks")
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
func (f CatTasks) WithContext(v context.Context) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.ctx = v
	}
}

// WithActions - a list of actions that should be returned. leave empty to return all..
func (f CatTasks) WithActions(v ...string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Actions = v
	}
}

// WithDetailed - return detailed task information (default: false).
func (f CatTasks) WithDetailed(v bool) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Detailed = &v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
func (f CatTasks) WithFormat(v string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Format = v
	}
}

// WithH - comma-separated list of column names to display.
func (f CatTasks) WithH(v ...string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
func (f CatTasks) WithHelp(v bool) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Help = &v
	}
}

// WithNodes - a list of node ids or names to limit the returned information; use `_local` to return information from the node you're connecting to, leave empty to get information from all nodes.
func (f CatTasks) WithNodes(v ...string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Nodes = v
	}
}

// WithParentTaskID - return tasks with specified parent task ID (node_id:task_number). set to -1 to return all..
func (f CatTasks) WithParentTaskID(v string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.ParentTaskID = v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
func (f CatTasks) WithS(v ...string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.S = v
	}
}

// WithTime - the unit in which to display time values.
func (f CatTasks) WithTime(v string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Time = v
	}
}

// WithTimeout - period to wait for a response. if no response is received before the timeout expires, the request fails and returns an error..
func (f CatTasks) WithTimeout(v time.Duration) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Timeout = v
	}
}

// WithV - verbose mode. display column headers.
func (f CatTasks) WithV(v bool) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.V = &v
	}
}

// WithWaitForCompletion - if `true`, the request blocks until the task has completed..
func (f CatTasks) WithWaitForCompletion(v bool) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CatTasks) WithPretty() func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CatTasks) WithHuman() func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CatTasks) WithErrorTrace() func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CatTasks) WithFilterPath(v ...string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CatTasks) WithHeader(h map[string]string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CatTasks) WithOpaqueID(s string) func(*CatTasksRequest) {
	return func(r *CatTasksRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
