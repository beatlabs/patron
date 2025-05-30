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
	"time"
)

func newMLCloseJobFunc(t Transport) MLCloseJob {
	return func(job_id string, o ...func(*MLCloseJobRequest)) (*Response, error) {
		var r = MLCloseJobRequest{JobID: job_id}
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

// MLCloseJob - Closes one or more anomaly detection jobs. A job can be opened and closed multiple times throughout its lifecycle.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-close-job.html.
type MLCloseJob func(job_id string, o ...func(*MLCloseJobRequest)) (*Response, error)

// MLCloseJobRequest configures the ML Close Job API request.
type MLCloseJobRequest struct {
	Body io.Reader

	JobID string

	AllowNoMatch *bool
	Force        *bool
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
func (r MLCloseJobRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.close_job")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("_close"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("anomaly_detectors")
	path.WriteString("/")
	path.WriteString(r.JobID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "job_id", r.JobID)
	}
	path.WriteString("/")
	path.WriteString("_close")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
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
		instrument.BeforeRequest(req, "ml.close_job")
		if reader := instrument.RecordRequestBody(ctx, "ml.close_job", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.close_job")
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
func (f MLCloseJob) WithContext(v context.Context) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.ctx = v
	}
}

// WithBody - The URL params optionally sent in the body.
func (f MLCloseJob) WithBody(v io.Reader) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.Body = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no jobs. (this includes `_all` string or when no jobs have been specified).
func (f MLCloseJob) WithAllowNoMatch(v bool) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.AllowNoMatch = &v
	}
}

// WithForce - true if the job should be forcefully closed.
func (f MLCloseJob) WithForce(v bool) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.Force = &v
	}
}

// WithTimeout - controls the time to wait until a job has closed. default to 30 minutes.
func (f MLCloseJob) WithTimeout(v time.Duration) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLCloseJob) WithPretty() func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLCloseJob) WithHuman() func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLCloseJob) WithErrorTrace() func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLCloseJob) WithFilterPath(v ...string) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLCloseJob) WithHeader(h map[string]string) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLCloseJob) WithOpaqueID(s string) func(*MLCloseJobRequest) {
	return func(r *MLCloseJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
