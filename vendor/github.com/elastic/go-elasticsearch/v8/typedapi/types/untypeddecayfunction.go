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
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/multivaluemode"
)

// UntypedDecayFunction type.
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/_types/query_dsl/compound.ts#L204-L207
type UntypedDecayFunction struct {
	DecayFunctionBase map[string]DecayPlacement `json:"-"`
	// MultiValueMode Determines how the distance is calculated when a field used for computing the
	// decay contains multiple values.
	MultiValueMode *multivaluemode.MultiValueMode `json:"multi_value_mode,omitempty"`
}

// MarhsalJSON overrides marshalling for types with additional properties
func (s UntypedDecayFunction) MarshalJSON() ([]byte, error) {
	type opt UntypedDecayFunction
	// We transform the struct to a map without the embedded additional properties map
	tmp := make(map[string]any, 0)

	data, err := json.Marshal(opt(s))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, err
	}

	// We inline the additional fields from the underlying map
	for key, value := range s.DecayFunctionBase {
		tmp[fmt.Sprintf("%s", key)] = value
	}
	delete(tmp, "DecayFunctionBase")

	data, err = json.Marshal(tmp)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// NewUntypedDecayFunction returns a UntypedDecayFunction.
func NewUntypedDecayFunction() *UntypedDecayFunction {
	r := &UntypedDecayFunction{
		DecayFunctionBase: make(map[string]DecayPlacement, 0),
	}

	return r
}
