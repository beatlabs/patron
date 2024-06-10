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
// https://github.com/elastic/elasticsearch-specification/tree/07bf82537a186562d8699685e3704ea338b268ef

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// RequestCacheStats type.
//
// https://github.com/elastic/elasticsearch-specification/blob/07bf82537a186562d8699685e3704ea338b268ef/specification/_types/Stats.ts#L244-L250
type RequestCacheStats struct {
	Evictions         int64   `json:"evictions"`
	HitCount          int64   `json:"hit_count"`
	MemorySize        *string `json:"memory_size,omitempty"`
	MemorySizeInBytes int64   `json:"memory_size_in_bytes"`
	MissCount         int64   `json:"miss_count"`
}

func (s *RequestCacheStats) UnmarshalJSON(data []byte) error {

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

		case "evictions":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("%s | %w", "Evictions", err)
				}
				s.Evictions = value
			case float64:
				f := int64(v)
				s.Evictions = f
			}

		case "hit_count":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("%s | %w", "HitCount", err)
				}
				s.HitCount = value
			case float64:
				f := int64(v)
				s.HitCount = f
			}

		case "memory_size":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "MemorySize", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.MemorySize = &o

		case "memory_size_in_bytes":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("%s | %w", "MemorySizeInBytes", err)
				}
				s.MemorySizeInBytes = value
			case float64:
				f := int64(v)
				s.MemorySizeInBytes = f
			}

		case "miss_count":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("%s | %w", "MissCount", err)
				}
				s.MissCount = value
			case float64:
				f := int64(v)
				s.MissCount = f
			}

		}
	}
	return nil
}

// NewRequestCacheStats returns a RequestCacheStats.
func NewRequestCacheStats() *RequestCacheStats {
	r := &RequestCacheStats{}

	return r
}
