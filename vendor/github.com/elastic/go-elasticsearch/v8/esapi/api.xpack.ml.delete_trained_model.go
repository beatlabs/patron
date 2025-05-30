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
	"time"
)

func newMLDeleteTrainedModelFunc(t Transport) MLDeleteTrainedModel {
	return func(model_id string, o ...func(*MLDeleteTrainedModelRequest)) (*Response, error) {
		var r = MLDeleteTrainedModelRequest{ModelID: model_id}
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

// MLDeleteTrainedModel - Deletes an existing trained inference model that is currently not referenced by an ingest pipeline.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-trained-models.html.
type MLDeleteTrainedModel func(model_id string, o ...func(*MLDeleteTrainedModelRequest)) (*Response, error)

// MLDeleteTrainedModelRequest configures the ML Delete Trained Model API request.
type MLDeleteTrainedModelRequest struct {
	ModelID string

	Force   *bool
	Timeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLDeleteTrainedModelRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.delete_trained_model")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "DELETE"

	path.Grow(7 + 1 + len("_ml") + 1 + len("trained_models") + 1 + len(r.ModelID))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("trained_models")
	path.WriteString("/")
	path.WriteString(r.ModelID)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "model_id", r.ModelID)
	}

	params = make(map[string]string)

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
		instrument.BeforeRequest(req, "ml.delete_trained_model")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.delete_trained_model")
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
func (f MLDeleteTrainedModel) WithContext(v context.Context) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.ctx = v
	}
}

// WithForce - true if the model should be forcefully deleted.
func (f MLDeleteTrainedModel) WithForce(v bool) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Force = &v
	}
}

// WithTimeout - controls the amount of time to wait for the model to be deleted..
func (f MLDeleteTrainedModel) WithTimeout(v time.Duration) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLDeleteTrainedModel) WithPretty() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLDeleteTrainedModel) WithHuman() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLDeleteTrainedModel) WithErrorTrace() func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLDeleteTrainedModel) WithFilterPath(v ...string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLDeleteTrainedModel) WithHeader(h map[string]string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLDeleteTrainedModel) WithOpaqueID(s string) func(*MLDeleteTrainedModelRequest) {
	return func(r *MLDeleteTrainedModelRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
