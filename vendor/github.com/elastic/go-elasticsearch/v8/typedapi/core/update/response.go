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
// https://github.com/elastic/elasticsearch-specification/tree/ac9c431ec04149d9048f2b8f9731e3c2f7f38754

package update

import (
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/result"
)

// Response holds the response body struct for the package update
//
// https://github.com/elastic/elasticsearch-specification/blob/ac9c431ec04149d9048f2b8f9731e3c2f7f38754/specification/_global/update/UpdateResponse.ts#L27-L29

type Response struct {
	ForcedRefresh *bool                 `json:"forced_refresh,omitempty"`
	Get           *types.InlineGet      `json:"get,omitempty"`
	Id_           string                `json:"_id"`
	Index_        string                `json:"_index"`
	PrimaryTerm_  int64                 `json:"_primary_term"`
	Result        result.Result         `json:"result"`
	SeqNo_        int64                 `json:"_seq_no"`
	Shards_       types.ShardStatistics `json:"_shards"`
	Version_      int64                 `json:"_version"`
}

// NewResponse returns a Response
func NewResponse() *Response {
	r := &Response{}
	return r
}
