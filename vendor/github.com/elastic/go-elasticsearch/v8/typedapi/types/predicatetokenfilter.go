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

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// PredicateTokenFilter type.
//
// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/_types/analysis/token_filters.ts#L297-L300
type PredicateTokenFilter struct {
	Script  Script  `json:"script"`
	Type    string  `json:"type,omitempty"`
	Version *string `json:"version,omitempty"`
}

func (s *PredicateTokenFilter) UnmarshalJSON(data []byte) error {

	dec := json.NewDecoder(bytes.NewReader(data))

	for {
		t, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		switch t {

		case "script":
			if err := dec.Decode(&s.Script); err != nil {
				return fmt.Errorf("%s | %w", "Script", err)
			}

		case "type":
			if err := dec.Decode(&s.Type); err != nil {
				return fmt.Errorf("%s | %w", "Type", err)
			}

		case "version":
			if err := dec.Decode(&s.Version); err != nil {
				return fmt.Errorf("%s | %w", "Version", err)
			}

		}
	}
	return nil
}

// MarshalJSON override marshalling to include literal value
func (s PredicateTokenFilter) MarshalJSON() ([]byte, error) {
	type innerPredicateTokenFilter PredicateTokenFilter
	tmp := innerPredicateTokenFilter{
		Script:  s.Script,
		Type:    s.Type,
		Version: s.Version,
	}

	tmp.Type = "predicate_token_filter"

	return json.Marshal(tmp)
}

// NewPredicateTokenFilter returns a PredicateTokenFilter.
func NewPredicateTokenFilter() *PredicateTokenFilter {
	r := &PredicateTokenFilter{}

	return r
}

// true

type PredicateTokenFilterVariant interface {
	PredicateTokenFilterCaster() *PredicateTokenFilter
}

func (s *PredicateTokenFilter) PredicateTokenFilterCaster() *PredicateTokenFilter {
	return s
}
