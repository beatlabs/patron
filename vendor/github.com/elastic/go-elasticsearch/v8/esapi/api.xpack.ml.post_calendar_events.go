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
// Code generated from specification version 8.1.0: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func newMLPostCalendarEventsFunc(t Transport) MLPostCalendarEvents {
	return func(calendar_id string, body io.Reader, o ...func(*MLPostCalendarEventsRequest)) (*Response, error) {
		var r = MLPostCalendarEventsRequest{CalendarID: calendar_id, Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLPostCalendarEvents - Posts scheduled events in a calendar.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-post-calendar-event.html.
//
type MLPostCalendarEvents func(calendar_id string, body io.Reader, o ...func(*MLPostCalendarEventsRequest)) (*Response, error)

// MLPostCalendarEventsRequest configures the ML Post Calendar Events API request.
//
type MLPostCalendarEventsRequest struct {
	Body io.Reader

	CalendarID string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLPostCalendarEventsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(7 + 1 + len("_ml") + 1 + len("calendars") + 1 + len(r.CalendarID) + 1 + len("events"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("calendars")
	path.WriteString("/")
	path.WriteString(r.CalendarID)
	path.WriteString("/")
	path.WriteString("events")

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
		return nil, err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if r.Body != nil {
		req.Header[headerContentType] = headerContentTypeJSON
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

	res, err := transport.Perform(req)
	if err != nil {
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
//
func (f MLPostCalendarEvents) WithContext(v context.Context) func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLPostCalendarEvents) WithPretty() func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLPostCalendarEvents) WithHuman() func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLPostCalendarEvents) WithErrorTrace() func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLPostCalendarEvents) WithFilterPath(v ...string) func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLPostCalendarEvents) WithHeader(h map[string]string) func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
//
func (f MLPostCalendarEvents) WithOpaqueID(s string) func(*MLPostCalendarEventsRequest) {
	return func(r *MLPostCalendarEventsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
