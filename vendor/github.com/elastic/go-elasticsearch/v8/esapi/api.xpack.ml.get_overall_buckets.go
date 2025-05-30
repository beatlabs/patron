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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newMLGetOverallBucketsFunc(t Transport) MLGetOverallBuckets {
	return func(job_id string, o ...func(*MLGetOverallBucketsRequest)) (*Response, error) {
		var r = MLGetOverallBucketsRequest{JobID: job_id}
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

// MLGetOverallBuckets - Retrieves overall bucket results that summarize the bucket results of multiple anomaly detection jobs.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-overall-buckets.html.
type MLGetOverallBuckets func(job_id string, o ...func(*MLGetOverallBucketsRequest)) (*Response, error)

// MLGetOverallBucketsRequest configures the ML Get Overall Buckets API request.
type MLGetOverallBucketsRequest struct {
	Body io.Reader

	JobID string

	AllowNoMatch   *bool
	BucketSpan     string
	End            string
	ExcludeInterim *bool
	OverallScore   interface{}
	Start          string
	TopN           *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetOverallBucketsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_overall_buckets")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("results") + 1 + len("overall_buckets"))
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
	path.WriteString("results")
	path.WriteString("/")
	path.WriteString("overall_buckets")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.BucketSpan != "" {
		params["bucket_span"] = r.BucketSpan
	}

	if r.End != "" {
		params["end"] = r.End
	}

	if r.ExcludeInterim != nil {
		params["exclude_interim"] = strconv.FormatBool(*r.ExcludeInterim)
	}

	if r.OverallScore != nil {
		params["overall_score"] = fmt.Sprintf("%v", r.OverallScore)
	}

	if r.Start != "" {
		params["start"] = r.Start
	}

	if r.TopN != nil {
		params["top_n"] = strconv.FormatInt(int64(*r.TopN), 10)
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
		instrument.BeforeRequest(req, "ml.get_overall_buckets")
		if reader := instrument.RecordRequestBody(ctx, "ml.get_overall_buckets", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_overall_buckets")
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
func (f MLGetOverallBuckets) WithContext(v context.Context) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.ctx = v
	}
}

// WithBody - Overall bucket selection details if not provided in URI.
func (f MLGetOverallBuckets) WithBody(v io.Reader) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.Body = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no jobs. (this includes `_all` string or when no jobs have been specified).
func (f MLGetOverallBuckets) WithAllowNoMatch(v bool) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithBucketSpan - the span of the overall buckets. defaults to the longest job bucket_span.
func (f MLGetOverallBuckets) WithBucketSpan(v string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.BucketSpan = v
	}
}

// WithEnd - returns overall buckets with timestamps earlier than this time.
func (f MLGetOverallBuckets) WithEnd(v string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.End = v
	}
}

// WithExcludeInterim - if true overall buckets that include interim buckets will be excluded.
func (f MLGetOverallBuckets) WithExcludeInterim(v bool) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.ExcludeInterim = &v
	}
}

// WithOverallScore - returns overall buckets with overall scores higher than this value.
func (f MLGetOverallBuckets) WithOverallScore(v interface{}) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.OverallScore = v
	}
}

// WithStart - returns overall buckets with timestamps after this time.
func (f MLGetOverallBuckets) WithStart(v string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.Start = v
	}
}

// WithTopN - the number of top job bucket scores to be used in the overall_score calculation.
func (f MLGetOverallBuckets) WithTopN(v int) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.TopN = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetOverallBuckets) WithPretty() func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetOverallBuckets) WithHuman() func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetOverallBuckets) WithErrorTrace() func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetOverallBuckets) WithFilterPath(v ...string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetOverallBuckets) WithHeader(h map[string]string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetOverallBuckets) WithOpaqueID(s string) func(*MLGetOverallBucketsRequest) {
	return func(r *MLGetOverallBucketsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
