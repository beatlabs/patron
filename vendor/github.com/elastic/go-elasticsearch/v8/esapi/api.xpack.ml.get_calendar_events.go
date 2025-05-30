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
	"net/http"
	"strconv"
	"strings"
)

func newMLGetCalendarEventsFunc(t Transport) MLGetCalendarEvents {
	return func(calendar_id string, o ...func(*MLGetCalendarEventsRequest)) (*Response, error) {
		var r = MLGetCalendarEventsRequest{CalendarID: calendar_id}
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

// MLGetCalendarEvents - Retrieves information about the scheduled events in calendars.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-calendar-event.html.
type MLGetCalendarEvents func(calendar_id string, o ...func(*MLGetCalendarEventsRequest)) (*Response, error)

// MLGetCalendarEventsRequest configures the ML Get Calendar Events API request.
type MLGetCalendarEventsRequest struct {
	CalendarID string

	End   interface{}
	From  *int
	JobID string
	Size  *int
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
func (r MLGetCalendarEventsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_calendar_events")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("calendars") + 1 + len(r.CalendarID) + 1 + len("events"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("calendars")
	path.WriteString("/")
	path.WriteString(r.CalendarID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "calendar_id", r.CalendarID)
	}
	path.WriteString("/")
	path.WriteString("events")

	params = make(map[string]string)

	if r.End != nil {
		params["end"] = fmt.Sprintf("%v", r.End)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.JobID != "" {
		params["job_id"] = r.JobID
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
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
		instrument.BeforeRequest(req, "ml.get_calendar_events")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_calendar_events")
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
func (f MLGetCalendarEvents) WithContext(v context.Context) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.ctx = v
	}
}

// WithEnd - get events before this time.
func (f MLGetCalendarEvents) WithEnd(v interface{}) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.End = v
	}
}

// WithFrom - skips a number of events.
func (f MLGetCalendarEvents) WithFrom(v int) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.From = &v
	}
}

// WithJobID - get events for the job. when this option is used calendar_id must be '_all'.
func (f MLGetCalendarEvents) WithJobID(v string) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.JobID = v
	}
}

// WithSize - specifies a max number of events to get.
func (f MLGetCalendarEvents) WithSize(v int) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.Size = &v
	}
}

// WithStart - get events after this time.
func (f MLGetCalendarEvents) WithStart(v string) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.Start = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetCalendarEvents) WithPretty() func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetCalendarEvents) WithHuman() func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetCalendarEvents) WithErrorTrace() func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetCalendarEvents) WithFilterPath(v ...string) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetCalendarEvents) WithHeader(h map[string]string) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetCalendarEvents) WithOpaqueID(s string) func(*MLGetCalendarEventsRequest) {
	return func(r *MLGetCalendarEventsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
