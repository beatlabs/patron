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

func newMLGetTrainedModelsStatsFunc(t Transport) MLGetTrainedModelsStats {
	return func(o ...func(*MLGetTrainedModelsStatsRequest)) (*Response, error) {
		var r = MLGetTrainedModelsStatsRequest{}
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

// MLGetTrainedModelsStats - Retrieves usage information for trained inference models.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-trained-models-stats.html.
type MLGetTrainedModelsStats func(o ...func(*MLGetTrainedModelsStatsRequest)) (*Response, error)

// MLGetTrainedModelsStatsRequest configures the ML Get Trained Models Stats API request.
type MLGetTrainedModelsStatsRequest struct {
	ModelID string

	AllowNoMatch *bool
	From         *int
	Size         *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetTrainedModelsStatsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_trained_models_stats")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("trained_models") + 1 + len(r.ModelID) + 1 + len("_stats"))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("trained_models")
	if r.ModelID != "" {
		path.WriteString("/")
		path.WriteString(r.ModelID)
		if instrument, ok := r.Instrument.(Instrumentation); ok {
			instrument.RecordPathPart(ctx, "model_id", r.ModelID)
		}
	}
	path.WriteString("/")
	path.WriteString("_stats")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

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
		instrument.BeforeRequest(req, "ml.get_trained_models_stats")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_trained_models_stats")
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
func (f MLGetTrainedModelsStats) WithContext(v context.Context) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.ctx = v
	}
}

// WithModelID - the ID of the trained models stats to fetch.
func (f MLGetTrainedModelsStats) WithModelID(v string) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.ModelID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no trained models. (this includes `_all` string or when no trained models have been specified).
func (f MLGetTrainedModelsStats) WithAllowNoMatch(v bool) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFrom - skips a number of trained models.
func (f MLGetTrainedModelsStats) WithFrom(v int) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.From = &v
	}
}

// WithSize - specifies a max number of trained models to get.
func (f MLGetTrainedModelsStats) WithSize(v int) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetTrainedModelsStats) WithPretty() func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetTrainedModelsStats) WithHuman() func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetTrainedModelsStats) WithErrorTrace() func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetTrainedModelsStats) WithFilterPath(v ...string) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetTrainedModelsStats) WithHeader(h map[string]string) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetTrainedModelsStats) WithOpaqueID(s string) func(*MLGetTrainedModelsStatsRequest) {
	return func(r *MLGetTrainedModelsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
