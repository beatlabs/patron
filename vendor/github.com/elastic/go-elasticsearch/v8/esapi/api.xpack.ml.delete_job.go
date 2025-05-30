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

func newMLDeleteJobFunc(t Transport) MLDeleteJob {
	return func(job_id string, o ...func(*MLDeleteJobRequest)) (*Response, error) {
		var r = MLDeleteJobRequest{JobID: job_id}
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

// MLDeleteJob - Deletes an existing anomaly detection job.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-delete-job.html.
type MLDeleteJob func(job_id string, o ...func(*MLDeleteJobRequest)) (*Response, error)

// MLDeleteJobRequest configures the ML Delete Job API request.
type MLDeleteJobRequest struct {
	JobID string

	DeleteUserAnnotations *bool
	Force                 *bool
	WaitForCompletion     *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLDeleteJobRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.delete_job")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "DELETE"

	path.Grow(7 + 1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID))
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

	params = make(map[string]string)

	if r.DeleteUserAnnotations != nil {
		params["delete_user_annotations"] = strconv.FormatBool(*r.DeleteUserAnnotations)
	}

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
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
		instrument.BeforeRequest(req, "ml.delete_job")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.delete_job")
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
func (f MLDeleteJob) WithContext(v context.Context) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.ctx = v
	}
}

// WithDeleteUserAnnotations - should annotations added by the user be deleted.
func (f MLDeleteJob) WithDeleteUserAnnotations(v bool) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.DeleteUserAnnotations = &v
	}
}

// WithForce - true if the job should be forcefully deleted.
func (f MLDeleteJob) WithForce(v bool) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.Force = &v
	}
}

// WithWaitForCompletion - should this request wait until the operation has completed before returning.
func (f MLDeleteJob) WithWaitForCompletion(v bool) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLDeleteJob) WithPretty() func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLDeleteJob) WithHuman() func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLDeleteJob) WithErrorTrace() func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLDeleteJob) WithFilterPath(v ...string) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLDeleteJob) WithHeader(h map[string]string) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLDeleteJob) WithOpaqueID(s string) func(*MLDeleteJobRequest) {
	return func(r *MLDeleteJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
