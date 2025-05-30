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
)

func newMLGetCalendarsFunc(t Transport) MLGetCalendars {
	return func(o ...func(*MLGetCalendarsRequest)) (*Response, error) {
		var r = MLGetCalendarsRequest{}
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

// MLGetCalendars - Retrieves configuration information for calendars.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-calendar.html.
type MLGetCalendars func(o ...func(*MLGetCalendarsRequest)) (*Response, error)

// MLGetCalendarsRequest configures the ML Get Calendars API request.
type MLGetCalendarsRequest struct {
	Body io.Reader

	CalendarID string

	From *int
	Size *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetCalendarsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_calendars")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("calendars") + 1 + len(r.CalendarID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("calendars")
	if r.CalendarID != "" {
		path.WriteString("/")
		path.WriteString(r.CalendarID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "calendar_id", r.CalendarID)
		}
	}

	params = make(map[string]string)

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
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
		instrument.BeforeRequest(req, "ml.get_calendars")
		if reader := instrument.RecordRequestBody(ctx, "ml.get_calendars", r.Body); reader != nil {
			req.Body = reader
		}
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_calendars")
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
func (f MLGetCalendars) WithContext(v context.Context) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.ctx = v
	}
}

// WithBody - The from and size parameters optionally sent in the body.
func (f MLGetCalendars) WithBody(v io.Reader) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.Body = v
	}
}

// WithCalendarID - the ID of the calendar to fetch.
func (f MLGetCalendars) WithCalendarID(v string) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.CalendarID = v
	}
}

// WithFrom - skips a number of calendars.
func (f MLGetCalendars) WithFrom(v int) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of calendars to get.
func (f MLGetCalendars) WithSize(v int) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetCalendars) WithPretty() func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetCalendars) WithHuman() func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetCalendars) WithErrorTrace() func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetCalendars) WithFilterPath(v ...string) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetCalendars) WithHeader(h map[string]string) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetCalendars) WithOpaqueID(s string) func(*MLGetCalendarsRequest) {
	return func(r *MLGetCalendarsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
