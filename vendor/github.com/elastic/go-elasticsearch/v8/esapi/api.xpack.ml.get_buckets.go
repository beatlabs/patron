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

func newMLGetBucketsFunc(t Transport) MLGetBuckets {
	return func(job_id string, o ...func(*MLGetBucketsRequest)) (*Response, error) {
		var r = MLGetBucketsRequest{JobID: job_id}
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

// MLGetBuckets - Retrieves anomaly detection job results for one or more buckets.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-bucket.html.
type MLGetBuckets func(job_id string, o ...func(*MLGetBucketsRequest)) (*Response, error)

// MLGetBucketsRequest configures the ML Get Buckets API request.
type MLGetBucketsRequest struct {
	Body io.Reader

	JobID     string
	Timestamp string

	AnomalyScore   interface{}
	Desc           *bool
	End            string
	ExcludeInterim *bool
	Expand         *bool
	From           *int
	Size           *int
	Sort           string
	Start          string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetBucketsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_buckets")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("results") + 1 + len("buckets") + 1 + len(r.Timestamp))
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
	path.WriteString("buckets")
	if r.Timestamp != "" {
		path.WriteString("/")
		path.WriteString(r.Timestamp)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "timestamp", r.Timestamp)
		}
	}

	params = make(map[string]string)

	if r.AnomalyScore != nil {
		params["anomaly_score"] = fmt.Sprintf("%v", r.AnomalyScore)
	}

	if r.Desc != nil {
		params["desc"] = strconv.FormatBool(*r.Desc)
	}

	if r.End != "" {
		params["end"] = r.End
	}

	if r.ExcludeInterim != nil {
		params["exclude_interim"] = strconv.FormatBool(*r.ExcludeInterim)
	}

	if r.Expand != nil {
		params["expand"] = strconv.FormatBool(*r.Expand)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if r.Sort != "" {
		params["sort"] = r.Sort
	}

	if r.Start != "" {
		params["start"] = r.Start
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
		instrument.BeforeRequest(req, "ml.get_buckets")
		if reader := instrument.RecordRequestBody(ctx, "ml.get_buckets", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_buckets")
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
func (f MLGetBuckets) WithContext(v context.Context) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.ctx = v
	}
}

// WithBody - Bucket selection details if not provided in URI.
func (f MLGetBuckets) WithBody(v io.Reader) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Body = v
	}
}

// WithTimestamp - the timestamp of the desired single bucket result.
func (f MLGetBuckets) WithTimestamp(v string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Timestamp = v
	}
}

// WithAnomalyScore - filter for the most anomalous buckets.
func (f MLGetBuckets) WithAnomalyScore(v interface{}) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.AnomalyScore = v
	}
}

// WithDesc - set the sort direction.
func (f MLGetBuckets) WithDesc(v bool) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Desc = &v
	}
}

// WithEnd - end time filter for buckets.
func (f MLGetBuckets) WithEnd(v string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.End = v
	}
}

// WithExcludeInterim - exclude interim results.
func (f MLGetBuckets) WithExcludeInterim(v bool) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.ExcludeInterim = &v
	}
}

// WithExpand - include anomaly records.
func (f MLGetBuckets) WithExpand(v bool) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Expand = &v
	}
}

// WithFrom - skips a number of buckets.
func (f MLGetBuckets) WithFrom(v int) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of buckets to get.
func (f MLGetBuckets) WithSize(v int) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Size = &v
	}
}

// WithSort - sort buckets by a particular field.
func (f MLGetBuckets) WithSort(v string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Sort = v
	}
}

// WithStart - start time filter for buckets.
func (f MLGetBuckets) WithStart(v string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Start = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetBuckets) WithPretty() func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetBuckets) WithHuman() func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetBuckets) WithErrorTrace() func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetBuckets) WithFilterPath(v ...string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetBuckets) WithHeader(h map[string]string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetBuckets) WithOpaqueID(s string) func(*MLGetBucketsRequest) {
	return func(r *MLGetBucketsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
