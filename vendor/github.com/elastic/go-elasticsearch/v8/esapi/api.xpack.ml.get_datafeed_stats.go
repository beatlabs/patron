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

func newMLGetDatafeedStatsFunc(t Transport) MLGetDatafeedStats {
	return func(o ...func(*MLGetDatafeedStatsRequest)) (*Response, error) {
		var r = MLGetDatafeedStatsRequest{}
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

// MLGetDatafeedStats - Retrieves usage information for datafeeds.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-datafeed-stats.html.
type MLGetDatafeedStats func(o ...func(*MLGetDatafeedStatsRequest)) (*Response, error)

// MLGetDatafeedStatsRequest configures the ML Get Datafeed Stats API request.
type MLGetDatafeedStatsRequest struct {
	DatafeedID string

	AllowNoMatch *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetDatafeedStatsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_datafeed_stats")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID) + 1 + len("_stats"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("datafeeds")
	if r.DatafeedID != "" {
		path.WriteString("/")
		path.WriteString(r.DatafeedID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "datafeed_id", r.DatafeedID)
		}
	}
	path.WriteString("/")
	path.WriteString("_stats")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
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
		instrument.BeforeRequest(req, "ml.get_datafeed_stats")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_datafeed_stats")
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
func (f MLGetDatafeedStats) WithContext(v context.Context) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.ctx = v
	}
}

// WithDatafeedID - the ID of the datafeeds stats to fetch.
func (f MLGetDatafeedStats) WithDatafeedID(v string) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.DatafeedID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
func (f MLGetDatafeedStats) WithAllowNoMatch(v bool) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetDatafeedStats) WithPretty() func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetDatafeedStats) WithHuman() func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetDatafeedStats) WithErrorTrace() func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetDatafeedStats) WithFilterPath(v ...string) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetDatafeedStats) WithHeader(h map[string]string) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetDatafeedStats) WithOpaqueID(s string) func(*MLGetDatafeedStatsRequest) {
	return func(r *MLGetDatafeedStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
