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
// https://github.com/elastic/elasticsearch-specification/tree/f6a370d0fba975752c644fc730f7c45610e28f36

package types

// IndexingSlowlogTresholds type.
//
// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/indices/_types/IndexSettings.ts#L595-L602
type IndexingSlowlogTresholds struct {
	// Index The indexing slow log, similar in functionality to the search slow log. The
	// log file name ends with `_index_indexing_slowlog.json`.
	// Log and the thresholds are configured in the same way as the search slowlog.
	Index *SlowlogTresholdLevels `json:"index,omitempty"`
}

// NewIndexingSlowlogTresholds returns a IndexingSlowlogTresholds.
func NewIndexingSlowlogTresholds() *IndexingSlowlogTresholds {
	r := &IndexingSlowlogTresholds{}

	return r
}

// true

type IndexingSlowlogTresholdsVariant interface {
	IndexingSlowlogTresholdsCaster() *IndexingSlowlogTresholds
}

func (s *IndexingSlowlogTresholds) IndexingSlowlogTresholdsCaster() *IndexingSlowlogTresholds {
	return s
}
