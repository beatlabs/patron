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

func newMLGetTrainedModelsFunc(t Transport) MLGetTrainedModels {
	return func(o ...func(*MLGetTrainedModelsRequest)) (*Response, error) {
		var r = MLGetTrainedModelsRequest{}
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

// MLGetTrainedModels - Retrieves configuration information for a trained inference model.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-trained-models.html.
type MLGetTrainedModels func(o ...func(*MLGetTrainedModelsRequest)) (*Response, error)

// MLGetTrainedModelsRequest configures the ML Get Trained Models API request.
type MLGetTrainedModelsRequest struct {
	ModelID string

	AllowNoMatch           *bool
	DecompressDefinition   *bool
	ExcludeGenerated       *bool
	From                   *int
	Include                string
	IncludeModelDefinition *bool
	Size                   *int
	Tags                   []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r MLGetTrainedModelsRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "ml.get_trained_models")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_ml") + 1 + len("trained_models") + 1 + len(r.ModelID))
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

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.DecompressDefinition != nil {
		params["decompress_definition"] = strconv.FormatBool(*r.DecompressDefinition)
	}

	if r.ExcludeGenerated != nil {
		params["exclude_generated"] = strconv.FormatBool(*r.ExcludeGenerated)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.Include != "" {
		params["include"] = r.Include
	}

	if r.IncludeModelDefinition != nil {
		params["include_model_definition"] = strconv.FormatBool(*r.IncludeModelDefinition)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if len(r.Tags) > 0 {
		params["tags"] = strings.Join(r.Tags, ",")
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
		instrument.BeforeRequest(req, "ml.get_trained_models")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "ml.get_trained_models")
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
func (f MLGetTrainedModels) WithContext(v context.Context) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.ctx = v
	}
}

// WithModelID - the ID of the trained models to fetch.
func (f MLGetTrainedModels) WithModelID(v string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.ModelID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no trained models. (this includes `_all` string or when no trained models have been specified).
func (f MLGetTrainedModels) WithAllowNoMatch(v bool) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithDecompressDefinition - should the model definition be decompressed into valid json or returned in a custom compressed format. defaults to true..
func (f MLGetTrainedModels) WithDecompressDefinition(v bool) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.DecompressDefinition = &v
	}
}

// WithExcludeGenerated - omits fields that are illegal to set on model put.
func (f MLGetTrainedModels) WithExcludeGenerated(v bool) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.ExcludeGenerated = &v
	}
}

// WithFrom - skips a number of trained models.
func (f MLGetTrainedModels) WithFrom(v int) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.From = &v
	}
}

// WithInclude - a comma-separate list of fields to optionally include. valid options are 'definition' and 'total_feature_importance'. default is none..
func (f MLGetTrainedModels) WithInclude(v string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.Include = v
	}
}

// WithIncludeModelDefinition - should the full model definition be included in the results. these definitions can be large. so be cautious when including them. defaults to false..
func (f MLGetTrainedModels) WithIncludeModelDefinition(v bool) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.IncludeModelDefinition = &v
	}
}

// WithSize - specifies a max number of trained models to get.
func (f MLGetTrainedModels) WithSize(v int) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.Size = &v
	}
}

// WithTags - a list of tags that the model must have..
func (f MLGetTrainedModels) WithTags(v ...string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.Tags = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetTrainedModels) WithPretty() func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetTrainedModels) WithHuman() func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetTrainedModels) WithErrorTrace() func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetTrainedModels) WithFilterPath(v ...string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetTrainedModels) WithHeader(h map[string]string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetTrainedModels) WithOpaqueID(s string) func(*MLGetTrainedModelsRequest) {
	return func(r *MLGetTrainedModelsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
