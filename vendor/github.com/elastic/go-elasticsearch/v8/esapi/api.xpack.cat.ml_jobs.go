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

func newCatMLJobsFunc(t Transport) CatMLJobs {
	return func(o ...func(*CatMLJobsRequest)) (*Response, error) {
		var r = CatMLJobsRequest{}
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

// CatMLJobs - Gets configuration and usage information about anomaly detection jobs.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/cat-anomaly-detectors.html.
type CatMLJobs func(o ...func(*CatMLJobsRequest)) (*Response, error)

// CatMLJobsRequest configures the CatML Jobs API request.
type CatMLJobsRequest struct {
	JobID string

	AllowNoMatch *bool
	Bytes        string
	Format       string
	H            []string
	Help         *bool
	S            []string
	Time         string
	V            *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r CatMLJobsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "cat.ml_jobs")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_cat") + 1 + len("ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("ml")
	path.WriteString("/")
	path.WriteString("anomaly_detectors")
	if r.JobID != "" {
		path.WriteString("/")
		path.WriteString(r.JobID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "job_id", r.JobID)
		}
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Bytes != "" {
		params["bytes"] = r.Bytes
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
		instrument.BeforeRequest(req, "cat.ml_jobs")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "cat.ml_jobs")
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
func (f CatMLJobs) WithContext(v context.Context) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.ctx = v
	}
}

// WithJobID - the ID of the jobs stats to fetch.
func (f CatMLJobs) WithJobID(v string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.JobID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no jobs. (this includes `_all` string or when no jobs have been specified).
func (f CatMLJobs) WithAllowNoMatch(v bool) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithBytes - the unit in which to display byte values.
func (f CatMLJobs) WithBytes(v string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Bytes = v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
func (f CatMLJobs) WithFormat(v string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Format = v
	}
}

// WithH - comma-separated list of column names to display.
func (f CatMLJobs) WithH(v ...string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
func (f CatMLJobs) WithHelp(v bool) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Help = &v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
func (f CatMLJobs) WithS(v ...string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.S = v
	}
}

// WithTime - the unit in which to display time values.
func (f CatMLJobs) WithTime(v string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
func (f CatMLJobs) WithV(v bool) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CatMLJobs) WithPretty() func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CatMLJobs) WithHuman() func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CatMLJobs) WithErrorTrace() func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CatMLJobs) WithFilterPath(v ...string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CatMLJobs) WithHeader(h map[string]string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CatMLJobs) WithOpaqueID(s string) func(*CatMLJobsRequest) {
	return func(r *CatMLJobsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
