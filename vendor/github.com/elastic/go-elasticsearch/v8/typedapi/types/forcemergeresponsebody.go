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
	"strconv"
)

// ForceMergeResponseBody type.
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/indices/forcemerge/_types/response.ts#L22-L28
type ForceMergeResponseBody struct {
	Shards_ *ShardStatistics `json:"_shards,omitempty"`
	// Task task contains a task id returned when wait_for_completion=false,
	// you can use the task_id to get the status of the task at _tasks/<task_id>
	Task *string `json:"task,omitempty"`
}

func (s *ForceMergeResponseBody) UnmarshalJSON(data []byte) error {

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

		case "_shards":
			if err := dec.Decode(&s.Shards_); err != nil {
				return fmt.Errorf("%s | %w", "Shards_", err)
			}

		case "task":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "Task", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.Task = &o

		}
	}
	return nil
}

// NewForceMergeResponseBody returns a ForceMergeResponseBody.
func NewForceMergeResponseBody() *ForceMergeResponseBody {
	r := &ForceMergeResponseBody{}

	return r
}
