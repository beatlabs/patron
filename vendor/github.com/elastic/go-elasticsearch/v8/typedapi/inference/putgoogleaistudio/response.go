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

package putgoogleaistudio

import (
	"encoding/json"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/tasktypegoogleaistudio"
)

// Response holds the response body struct for the package putgoogleaistudio
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/inference/put_googleaistudio/PutGoogleAiStudioResponse.ts#L22-L25
type Response struct {

	// ChunkingSettings Chunking configuration object
	ChunkingSettings *types.InferenceChunkingSettings `json:"chunking_settings,omitempty"`
	// InferenceId The inference Id
	InferenceId string `json:"inference_id"`
	// Service The service type
	Service string `json:"service"`
	// ServiceSettings Settings specific to the service
	ServiceSettings json.RawMessage `json:"service_settings"`
	// TaskSettings Task settings specific to the service and task type
	TaskSettings json.RawMessage `json:"task_settings,omitempty"`
	// TaskType The task type
	TaskType tasktypegoogleaistudio.TaskTypeGoogleAIStudio `json:"task_type"`
}

// NewResponse returns a Response
func NewResponse() *Response {
	r := &Response{}
	return r
}
