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

func newNodesInfoFunc(t Transport) NodesInfo {
	return func(o ...func(*NodesInfoRequest)) (*Response, error) {
		var r = NodesInfoRequest{}
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

// NodesInfo returns information about nodes in the cluster.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/cluster-nodes-info.html.
type NodesInfo func(o ...func(*NodesInfoRequest)) (*Response, error)

// NodesInfoRequest configures the Nodes Info API request.
type NodesInfoRequest struct {
	Metric []string
	NodeID []string

	FlatSettings *bool
	Timeout      time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r NodesInfoRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "nodes.info")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_nodes") + 1 + len(strings.Join(r.NodeID, ",")) + 1 + len(strings.Join(r.Metric, ",")))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_nodes")
	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.NodeID, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "node_id", strings.Join(r.NodeID, ","))
		}
	}
	if len(r.Metric) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Metric, ","))
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "metric", strings.Join(r.Metric, ","))
		}
	}

	params = make(map[string]string)

	if r.FlatSettings != nil {
		params["flat_settings"] = strconv.FormatBool(*r.FlatSettings)
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
		instrument.BeforeRequest(req, "nodes.info")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "nodes.info")
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
func (f NodesInfo) WithContext(v context.Context) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.ctx = v
	}
}

// WithMetric - a list of metrics you wish returned. leave empty to return all metrics..
func (f NodesInfo) WithMetric(v ...string) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.Metric = v
	}
}

// WithNodeID - a list of node ids or names to limit the returned information; use `_local` to return information from the node you're connecting to, leave empty to get information from all nodes.
func (f NodesInfo) WithNodeID(v ...string) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.NodeID = v
	}
}

// WithFlatSettings - return settings in flat format (default: false).
func (f NodesInfo) WithFlatSettings(v bool) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.FlatSettings = &v
	}
}

// WithTimeout - explicit operation timeout.
func (f NodesInfo) WithTimeout(v time.Duration) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f NodesInfo) WithPretty() func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f NodesInfo) WithHuman() func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f NodesInfo) WithErrorTrace() func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f NodesInfo) WithFilterPath(v ...string) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f NodesInfo) WithHeader(h map[string]string) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f NodesInfo) WithOpaqueID(s string) func(*NodesInfoRequest) {
	return func(r *NodesInfoRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
