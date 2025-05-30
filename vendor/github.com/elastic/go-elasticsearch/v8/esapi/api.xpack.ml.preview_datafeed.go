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
	"strings"
)

func newMLPreviewDatafeedFunc(t Transport) MLPreviewDatafeed {
	return func(o ...func(*MLPreviewDatafeedRequest)) (*Response, error) {
		var r = MLPreviewDatafeedRequest{}
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

// MLPreviewDatafeed - Previews a datafeed.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-preview-datafeed.html.
type MLPreviewDatafeed func(o ...func(*MLPreviewDatafeedRequest)) (*Response, error)

// MLPreviewDatafeedRequest configures the ML Preview Datafeed API request.
type MLPreviewDatafeedRequest struct {
	Body io.Reader

	DatafeedID string

	End   string
	Start string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLPreviewDatafeedRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.preview_datafeed")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID) + 1 + len("_preview"))
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
	path.WriteString("_preview")

	params = make(map[string]string)

	if r.End != "" {
		params["end"] = r.End
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
		instrument.BeforeRequest(req, "ml.preview_datafeed")
		if reader := instrument.RecordRequestBody(ctx, "ml.preview_datafeed", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.preview_datafeed")
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
func (f MLPreviewDatafeed) WithContext(v context.Context) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.ctx = v
	}
}

// WithBody - The datafeed config and job config with which to execute the preview.
func (f MLPreviewDatafeed) WithBody(v io.Reader) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.Body = v
	}
}

// WithDatafeedID - the ID of the datafeed to preview.
func (f MLPreviewDatafeed) WithDatafeedID(v string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.DatafeedID = v
	}
}

// WithEnd - the end time when the datafeed preview should stop.
func (f MLPreviewDatafeed) WithEnd(v string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.End = v
	}
}

// WithStart - the start time from where the datafeed preview should begin.
func (f MLPreviewDatafeed) WithStart(v string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.Start = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLPreviewDatafeed) WithPretty() func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLPreviewDatafeed) WithHuman() func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLPreviewDatafeed) WithErrorTrace() func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLPreviewDatafeed) WithFilterPath(v ...string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLPreviewDatafeed) WithHeader(h map[string]string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLPreviewDatafeed) WithOpaqueID(s string) func(*MLPreviewDatafeedRequest) {
	return func(r *MLPreviewDatafeedRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
