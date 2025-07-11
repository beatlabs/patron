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
// https://github.com/elastic/elasticsearch-specification/tree/3a94b6715915b1e9311724a2614c643368eece90

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// QueryRuleset type.
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/query_rules/_types/QueryRuleset.ts#L25-L34
type QueryRuleset struct {
	// Rules Rules associated with the query ruleset.
	Rules []QueryRule `json:"rules"`
	// RulesetId A unique identifier for the ruleset.
	RulesetId string `json:"ruleset_id"`
}

func (s *QueryRuleset) UnmarshalJSON(data []byte) error {

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

		case "rules":
			if err := dec.Decode(&s.Rules); err != nil {
				return fmt.Errorf("%s | %w", "Rules", err)
			}

		case "ruleset_id":
			if err := dec.Decode(&s.RulesetId); err != nil {
				return fmt.Errorf("%s | %w", "RulesetId", err)
			}

		}
	}
	return nil
}

// NewQueryRuleset returns a QueryRuleset.
func NewQueryRuleset() *QueryRuleset {
	r := &QueryRuleset{}

	return r
}
