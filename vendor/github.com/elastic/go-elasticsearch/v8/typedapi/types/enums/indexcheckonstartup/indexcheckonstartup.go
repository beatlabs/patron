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

// Package indexcheckonstartup
package indexcheckonstartup

import "strings"

// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/indices/_types/IndexSettings.ts#L270-L277
type IndexCheckOnStartup struct {
	Name string
}

var (
	True = IndexCheckOnStartup{"true"}

	False = IndexCheckOnStartup{"false"}

	Checksum = IndexCheckOnStartup{"checksum"}
)

func (i *IndexCheckOnStartup) UnmarshalJSON(data []byte) error {
	return i.UnmarshalText(data)
}

func (i IndexCheckOnStartup) MarshalText() (text []byte, err error) {
	return []byte(i.String()), nil
}

func (i *IndexCheckOnStartup) UnmarshalText(text []byte) error {
	switch strings.ReplaceAll(strings.ToLower(string(text)), "\"", "") {

	case "true":
		*i = True
	case "false":
		*i = False
	case "checksum":
		*i = Checksum
	default:
		*i = IndexCheckOnStartup{string(text)}
	}

	return nil
}

func (i IndexCheckOnStartup) String() string {
	return i.Name
}
