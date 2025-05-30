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
	"strconv"
)

// DictionaryDecompounderTokenFilter type.
//
// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/_types/analysis/token_filters.ts#L53-L55
type DictionaryDecompounderTokenFilter struct {
	HyphenationPatternsPath *string  `json:"hyphenation_patterns_path,omitempty"`
	MaxSubwordSize          *int     `json:"max_subword_size,omitempty"`
	MinSubwordSize          *int     `json:"min_subword_size,omitempty"`
	MinWordSize             *int     `json:"min_word_size,omitempty"`
	OnlyLongestMatch        *bool    `json:"only_longest_match,omitempty"`
	Type                    string   `json:"type,omitempty"`
	Version                 *string  `json:"version,omitempty"`
	WordList                []string `json:"word_list,omitempty"`
	WordListPath            *string  `json:"word_list_path,omitempty"`
}

func (s *DictionaryDecompounderTokenFilter) UnmarshalJSON(data []byte) error {

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

		case "hyphenation_patterns_path":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "HyphenationPatternsPath", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.HyphenationPatternsPath = &o

		case "max_subword_size":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "MaxSubwordSize", err)
				}
				s.MaxSubwordSize = &value
			case float64:
				f := int(v)
				s.MaxSubwordSize = &f
			}

		case "min_subword_size":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "MinSubwordSize", err)
				}
				s.MinSubwordSize = &value
			case float64:
				f := int(v)
				s.MinSubwordSize = &f
			}

		case "min_word_size":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "MinWordSize", err)
				}
				s.MinWordSize = &value
			case float64:
				f := int(v)
				s.MinWordSize = &f
			}

		case "only_longest_match":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseBool(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "OnlyLongestMatch", err)
				}
				s.OnlyLongestMatch = &value
			case bool:
				s.OnlyLongestMatch = &v
			}

		case "type":
			if err := dec.Decode(&s.Type); err != nil {
				return fmt.Errorf("%s | %w", "Type", err)
			}

		case "version":
			if err := dec.Decode(&s.Version); err != nil {
				return fmt.Errorf("%s | %w", "Version", err)
			}

		case "word_list":
			if err := dec.Decode(&s.WordList); err != nil {
				return fmt.Errorf("%s | %w", "WordList", err)
			}

		case "word_list_path":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "WordListPath", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.WordListPath = &o

		}
	}
	return nil
}

// MarshalJSON override marshalling to include literal value
func (s DictionaryDecompounderTokenFilter) MarshalJSON() ([]byte, error) {
	type innerDictionaryDecompounderTokenFilter DictionaryDecompounderTokenFilter
	tmp := innerDictionaryDecompounderTokenFilter{
		HyphenationPatternsPath: s.HyphenationPatternsPath,
		MaxSubwordSize:          s.MaxSubwordSize,
		MinSubwordSize:          s.MinSubwordSize,
		MinWordSize:             s.MinWordSize,
		OnlyLongestMatch:        s.OnlyLongestMatch,
		Type:                    s.Type,
		Version:                 s.Version,
		WordList:                s.WordList,
		WordListPath:            s.WordListPath,
	}

	tmp.Type = "dictionary_decompounder"

	return json.Marshal(tmp)
}

// NewDictionaryDecompounderTokenFilter returns a DictionaryDecompounderTokenFilter.
func NewDictionaryDecompounderTokenFilter() *DictionaryDecompounderTokenFilter {
	r := &DictionaryDecompounderTokenFilter{}

	return r
}

// true

type DictionaryDecompounderTokenFilterVariant interface {
	DictionaryDecompounderTokenFilterCaster() *DictionaryDecompounderTokenFilter
}

func (s *DictionaryDecompounderTokenFilter) DictionaryDecompounderTokenFilterCaster() *DictionaryDecompounderTokenFilter {
	return s
}
