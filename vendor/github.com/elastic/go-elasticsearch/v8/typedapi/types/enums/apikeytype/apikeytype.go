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
// https://github.com/elastic/elasticsearch-specification/tree/2f823ff6fcaa7f3f0f9b990dc90512d8901e5d64

// Package apikeytype
package apikeytype

import "strings"

// https://github.com/elastic/elasticsearch-specification/blob/2f823ff6fcaa7f3f0f9b990dc90512d8901e5d64/specification/security/_types/ApiKey.ts#L115-L118
type ApiKeyType struct {
	Name string
}

var (
	Rest = ApiKeyType{"rest"}

	Crosscluster = ApiKeyType{"cross_cluster"}
)

func (a ApiKeyType) MarshalText() (text []byte, err error) {
	return []byte(a.String()), nil
}

func (a *ApiKeyType) UnmarshalText(text []byte) error {
	switch strings.ReplaceAll(strings.ToLower(string(text)), "\"", "") {

	case "rest":
		*a = Rest
	case "cross_cluster":
		*a = Crosscluster
	default:
		*a = ApiKeyType{string(text)}
	}

	return nil
}

func (a ApiKeyType) String() string {
	return a.Name
}