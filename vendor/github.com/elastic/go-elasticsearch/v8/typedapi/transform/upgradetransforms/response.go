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

// Code generated from the elasticsearch-specification DO NOT EDIT.
// https://github.com/elastic/elasticsearch-specification/tree/5bf86339cd4bda77d07f6eaa6789b72f9c0279b1

package upgradetransforms

// Response holds the response body struct for the package upgradetransforms
//
// https://github.com/elastic/elasticsearch-specification/blob/5bf86339cd4bda77d07f6eaa6789b72f9c0279b1/specification/transform/upgrade_transforms/UpgradeTransformsResponse.ts#L25-L34
type Response struct {

	// NeedsUpdate The number of transforms that need to be upgraded.
	NeedsUpdate int `json:"needs_update"`
	// NoAction The number of transforms that don’t require upgrading.
	NoAction int `json:"no_action"`
	// Updated The number of transforms that have been upgraded.
	Updated int `json:"updated"`
}

// NewResponse returns a Response
func NewResponse() *Response {
	r := &Response{}
	return r
}
