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

func newMLGetDataFrameAnalyticsStatsFunc(t Transport) MLGetDataFrameAnalyticsStats {
	return func(o ...func(*MLGetDataFrameAnalyticsStatsRequest)) (*Response, error) {
		var r = MLGetDataFrameAnalyticsStatsRequest{}
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

// MLGetDataFrameAnalyticsStats - Retrieves usage information for data frame analytics jobs.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-dfanalytics-stats.html.
type MLGetDataFrameAnalyticsStats func(o ...func(*MLGetDataFrameAnalyticsStatsRequest)) (*Response, error)

// MLGetDataFrameAnalyticsStatsRequest configures the ML Get Data Frame Analytics Stats API request.
type MLGetDataFrameAnalyticsStatsRequest struct {
	ID string

	AllowNoMatch *bool
	From         *int
	Size         *int
	Verbose      *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetDataFrameAnalyticsStatsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_data_frame_analytics_stats")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.ID) + 1 + len("_stats"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	if r.ID != "" {
		path.WriteString("/")
		path.WriteString(r.ID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "id", r.ID)
		}
	}
	path.WriteString("/")
	path.WriteString("_stats")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if r.Verbose != nil {
		params["verbose"] = strconv.FormatBool(*r.Verbose)
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
		instrument.BeforeRequest(req, "ml.get_data_frame_analytics_stats")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_data_frame_analytics_stats")
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
func (f MLGetDataFrameAnalyticsStats) WithContext(v context.Context) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.ctx = v
	}
}

// WithID - the ID of the data frame analytics stats to fetch.
func (f MLGetDataFrameAnalyticsStats) WithID(v string) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.ID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no data frame analytics. (this includes `_all` string or when no data frame analytics have been specified).
func (f MLGetDataFrameAnalyticsStats) WithAllowNoMatch(v bool) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFrom - skips a number of analytics.
func (f MLGetDataFrameAnalyticsStats) WithFrom(v int) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of analytics to get.
func (f MLGetDataFrameAnalyticsStats) WithSize(v int) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.Size = &v
	}
}

// WithVerbose - whether the stats response should be verbose.
func (f MLGetDataFrameAnalyticsStats) WithVerbose(v bool) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.Verbose = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetDataFrameAnalyticsStats) WithPretty() func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetDataFrameAnalyticsStats) WithHuman() func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetDataFrameAnalyticsStats) WithErrorTrace() func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetDataFrameAnalyticsStats) WithFilterPath(v ...string) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetDataFrameAnalyticsStats) WithHeader(h map[string]string) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetDataFrameAnalyticsStats) WithOpaqueID(s string) func(*MLGetDataFrameAnalyticsStatsRequest) {
	return func(r *MLGetDataFrameAnalyticsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
