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

func newMLExplainDataFrameAnalyticsFunc(t Transport) MLExplainDataFrameAnalytics {
	return func(o ...func(*MLExplainDataFrameAnalyticsRequest)) (*Response, error) {
		var r = MLExplainDataFrameAnalyticsRequest{}
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

// MLExplainDataFrameAnalytics - Explains a data frame analytics config.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/explain-dfanalytics.html.
type MLExplainDataFrameAnalytics func(o ...func(*MLExplainDataFrameAnalyticsRequest)) (*Response, error)

// MLExplainDataFrameAnalyticsRequest configures the ML Explain Data Frame Analytics API request.
type MLExplainDataFrameAnalyticsRequest struct {
	DocumentID string

	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLExplainDataFrameAnalyticsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.explain_data_frame_analytics")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.DocumentID) + 1 + len("_explain"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "id", r.DocumentID)
		}
	}
	path.WriteString("/")
	path.WriteString("_explain")

	params = make(map[string]string)

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
		instrument.BeforeRequest(req, "ml.explain_data_frame_analytics")
		if reader := instrument.RecordRequestBody(ctx, "ml.explain_data_frame_analytics", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.explain_data_frame_analytics")
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
func (f MLExplainDataFrameAnalytics) WithContext(v context.Context) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.ctx = v
	}
}

// WithBody - The data frame analytics config to explain.
func (f MLExplainDataFrameAnalytics) WithBody(v io.Reader) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Body = v
	}
}

// WithDocumentID - the ID of the data frame analytics to explain.
func (f MLExplainDataFrameAnalytics) WithDocumentID(v string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.DocumentID = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLExplainDataFrameAnalytics) WithPretty() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLExplainDataFrameAnalytics) WithHuman() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLExplainDataFrameAnalytics) WithErrorTrace() func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLExplainDataFrameAnalytics) WithFilterPath(v ...string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLExplainDataFrameAnalytics) WithHeader(h map[string]string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLExplainDataFrameAnalytics) WithOpaqueID(s string) func(*MLExplainDataFrameAnalyticsRequest) {
	return func(r *MLExplainDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
