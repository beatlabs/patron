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

func newCatMLDatafeedsFunc(t Transport) CatMLDatafeeds {
	return func(o ...func(*CatMLDatafeedsRequest)) (*Response, error) {
		var r = CatMLDatafeedsRequest{}
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

// CatMLDatafeeds - Gets configuration and usage information about datafeeds.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/cat-datafeeds.html.
type CatMLDatafeeds func(o ...func(*CatMLDatafeedsRequest)) (*Response, error)

// CatMLDatafeedsRequest configures the CatML Datafeeds API request.
type CatMLDatafeedsRequest struct {
	DatafeedID string

	AllowNoMatch *bool
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
func (r CatMLDatafeedsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "cat.ml_datafeeds")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_cat") + 1 + len("ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("ml")
	path.WriteString("/")
	path.WriteString("datafeeds")
	if r.DatafeedID != "" {
		path.WriteString("/")
		path.WriteString(r.DatafeedID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "datafeed_id", r.DatafeedID)
		}
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
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
		instrument.BeforeRequest(req, "cat.ml_datafeeds")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "cat.ml_datafeeds")
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
func (f CatMLDatafeeds) WithContext(v context.Context) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.ctx = v
	}
}

// WithDatafeedID - the ID of the datafeeds stats to fetch.
func (f CatMLDatafeeds) WithDatafeedID(v string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.DatafeedID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
func (f CatMLDatafeeds) WithAllowNoMatch(v bool) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
func (f CatMLDatafeeds) WithFormat(v string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.Format = v
	}
}

// WithH - comma-separated list of column names to display.
func (f CatMLDatafeeds) WithH(v ...string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
func (f CatMLDatafeeds) WithHelp(v bool) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.Help = &v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
func (f CatMLDatafeeds) WithS(v ...string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.S = v
	}
}

// WithTime - the unit in which to display time values.
func (f CatMLDatafeeds) WithTime(v string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
func (f CatMLDatafeeds) WithV(v bool) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CatMLDatafeeds) WithPretty() func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CatMLDatafeeds) WithHuman() func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CatMLDatafeeds) WithErrorTrace() func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CatMLDatafeeds) WithFilterPath(v ...string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CatMLDatafeeds) WithHeader(h map[string]string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CatMLDatafeeds) WithOpaqueID(s string) func(*CatMLDatafeedsRequest) {
	return func(r *CatMLDatafeedsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
