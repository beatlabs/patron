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
	"strings"
)

func newSecuritySamlServiceProviderMetadataFunc(t Transport) SecuritySamlServiceProviderMetadata {
	return func(realm_name string, o ...func(*SecuritySamlServiceProviderMetadataRequest)) (*Response, error) {
		var r = SecuritySamlServiceProviderMetadataRequest{RealmName: realm_name}
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

// SecuritySamlServiceProviderMetadata - Generates SAML metadata for the Elastic stack SAML 2.0 Service Provider
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-saml-sp-metadata.html.
type SecuritySamlServiceProviderMetadata func(realm_name string, o ...func(*SecuritySamlServiceProviderMetadataRequest)) (*Response, error)

// SecuritySamlServiceProviderMetadataRequest configures the Security Saml Service Provider Metadata API request.
type SecuritySamlServiceProviderMetadataRequest struct {
	RealmName string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context

	Instrument Instrumentation
}

// Do executes the request and returns response or error.
func (r SecuritySamlServiceProviderMetadataRequest) Do(providedCtx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
		ctx    context.Context
	)

	if instrument, ok := r.Instrument.(Instrumentation); ok {
		ctx = instrument.Start(providedCtx, "security.saml_service_provider_metadata")
		defer instrument.Close(ctx)
	}
	if ctx == nil {
		ctx = providedCtx
	}

	method = "GET"

	path.Grow(7 + 1 + len("_security") + 1 + len("saml") + 1 + len("metadata") + 1 + len(r.RealmName))
	path.WriteString("http://")
	path.WriteString("/")
	path.WriteString("_security")
	path.WriteString("/")
	path.WriteString("saml")
	path.WriteString("/")
	path.WriteString("metadata")
	path.WriteString("/")
	path.WriteString(r.RealmName)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.RecordPathPart(ctx, "realm_name", r.RealmName)
	}

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
		instrument.BeforeRequest(req, "security.saml_service_provider_metadata")
	}
	res, err := transport.Perform(req)
	if instrument, ok := r.Instrument.(Instrumentation); ok {
		instrument.AfterRequest(req, "elasticsearch", "security.saml_service_provider_metadata")
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
func (f SecuritySamlServiceProviderMetadata) WithContext(v context.Context) func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SecuritySamlServiceProviderMetadata) WithPretty() func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SecuritySamlServiceProviderMetadata) WithHuman() func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SecuritySamlServiceProviderMetadata) WithErrorTrace() func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SecuritySamlServiceProviderMetadata) WithFilterPath(v ...string) func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SecuritySamlServiceProviderMetadata) WithHeader(h map[string]string) func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SecuritySamlServiceProviderMetadata) WithOpaqueID(s string) func(*SecuritySamlServiceProviderMetadataRequest) {
	return func(r *SecuritySamlServiceProviderMetadataRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
