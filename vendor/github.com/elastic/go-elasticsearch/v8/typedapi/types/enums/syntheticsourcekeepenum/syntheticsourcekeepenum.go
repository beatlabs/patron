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

// Package syntheticsourcekeepenum
package syntheticsourcekeepenum

import "strings"

// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/_types/mapping/Property.ts#L99-L117
type SyntheticSourceKeepEnum struct {
	Name string
}

var (
	None = SyntheticSourceKeepEnum{"none"}

	Arrays = SyntheticSourceKeepEnum{"arrays"}

	All = SyntheticSourceKeepEnum{"all"}
)

func (s SyntheticSourceKeepEnum) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

func (s *SyntheticSourceKeepEnum) UnmarshalText(text []byte) error {
	switch strings.ReplaceAll(strings.ToLower(string(text)), "\"", "") {

	case "none":
		*s = None
	case "arrays":
		*s = Arrays
	case "all":
		*s = All
	default:
		*s = SyntheticSourceKeepEnum{string(text)}
	}

	return nil
}

func (s SyntheticSourceKeepEnum) String() string {
	return s.Name
}
