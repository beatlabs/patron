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

	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/healthstatus"
)

// HealthResponseBody type.
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/cluster/health/ClusterHealthResponse.ts#L39-L76
type HealthResponseBody struct {
	// ActivePrimaryShards The number of active primary shards.
	ActivePrimaryShards int `json:"active_primary_shards"`
	// ActiveShards The total number of active primary and replica shards.
	ActiveShards int `json:"active_shards"`
	// ActiveShardsPercent The ratio of active shards in the cluster expressed as a string formatted
	// percentage.
	ActiveShardsPercent *string `json:"active_shards_percent,omitempty"`
	// ActiveShardsPercentAsNumber The ratio of active shards in the cluster expressed as a percentage.
	ActiveShardsPercentAsNumber Float64 `json:"active_shards_percent_as_number"`
	// ClusterName The name of the cluster.
	ClusterName string `json:"cluster_name"`
	// DelayedUnassignedShards The number of shards whose allocation has been delayed by the timeout
	// settings.
	DelayedUnassignedShards int                         `json:"delayed_unassigned_shards"`
	Indices                 map[string]IndexHealthStats `json:"indices,omitempty"`
	// InitializingShards The number of shards that are under initialization.
	InitializingShards int `json:"initializing_shards"`
	// NumberOfDataNodes The number of nodes that are dedicated data nodes.
	NumberOfDataNodes int `json:"number_of_data_nodes"`
	// NumberOfInFlightFetch The number of unfinished fetches.
	NumberOfInFlightFetch int `json:"number_of_in_flight_fetch"`
	// NumberOfNodes The number of nodes within the cluster.
	NumberOfNodes int `json:"number_of_nodes"`
	// NumberOfPendingTasks The number of cluster-level changes that have not yet been executed.
	NumberOfPendingTasks int `json:"number_of_pending_tasks"`
	// RelocatingShards The number of shards that are under relocation.
	RelocatingShards int                       `json:"relocating_shards"`
	Status           healthstatus.HealthStatus `json:"status"`
	// TaskMaxWaitingInQueue The time since the earliest initiated task is waiting for being performed.
	TaskMaxWaitingInQueue Duration `json:"task_max_waiting_in_queue,omitempty"`
	// TaskMaxWaitingInQueueMillis The time expressed in milliseconds since the earliest initiated task is
	// waiting for being performed.
	TaskMaxWaitingInQueueMillis int64 `json:"task_max_waiting_in_queue_millis"`
	// TimedOut If false the response returned within the period of time that is specified by
	// the timeout parameter (30s by default)
	TimedOut bool `json:"timed_out"`
	// UnassignedPrimaryShards The number of primary shards that are not allocated.
	UnassignedPrimaryShards int `json:"unassigned_primary_shards"`
	// UnassignedShards The number of shards that are not allocated.
	UnassignedShards int `json:"unassigned_shards"`
}

func (s *HealthResponseBody) UnmarshalJSON(data []byte) error {

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

		case "active_primary_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "ActivePrimaryShards", err)
				}
				s.ActivePrimaryShards = value
			case float64:
				f := int(v)
				s.ActivePrimaryShards = f
			}

		case "active_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "ActiveShards", err)
				}
				s.ActiveShards = value
			case float64:
				f := int(v)
				s.ActiveShards = f
			}

		case "active_shards_percent":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "ActiveShardsPercent", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.ActiveShardsPercent = &o

		case "active_shards_percent_as_number":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return fmt.Errorf("%s | %w", "ActiveShardsPercentAsNumber", err)
				}
				f := Float64(value)
				s.ActiveShardsPercentAsNumber = f
			case float64:
				f := Float64(v)
				s.ActiveShardsPercentAsNumber = f
			}

		case "cluster_name":
			if err := dec.Decode(&s.ClusterName); err != nil {
				return fmt.Errorf("%s | %w", "ClusterName", err)
			}

		case "delayed_unassigned_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "DelayedUnassignedShards", err)
				}
				s.DelayedUnassignedShards = value
			case float64:
				f := int(v)
				s.DelayedUnassignedShards = f
			}

		case "indices":
			if s.Indices == nil {
				s.Indices = make(map[string]IndexHealthStats, 0)
			}
			if err := dec.Decode(&s.Indices); err != nil {
				return fmt.Errorf("%s | %w", "Indices", err)
			}

		case "initializing_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "InitializingShards", err)
				}
				s.InitializingShards = value
			case float64:
				f := int(v)
				s.InitializingShards = f
			}

		case "number_of_data_nodes":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "NumberOfDataNodes", err)
				}
				s.NumberOfDataNodes = value
			case float64:
				f := int(v)
				s.NumberOfDataNodes = f
			}

		case "number_of_in_flight_fetch":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "NumberOfInFlightFetch", err)
				}
				s.NumberOfInFlightFetch = value
			case float64:
				f := int(v)
				s.NumberOfInFlightFetch = f
			}

		case "number_of_nodes":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "NumberOfNodes", err)
				}
				s.NumberOfNodes = value
			case float64:
				f := int(v)
				s.NumberOfNodes = f
			}

		case "number_of_pending_tasks":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "NumberOfPendingTasks", err)
				}
				s.NumberOfPendingTasks = value
			case float64:
				f := int(v)
				s.NumberOfPendingTasks = f
			}

		case "relocating_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "RelocatingShards", err)
				}
				s.RelocatingShards = value
			case float64:
				f := int(v)
				s.RelocatingShards = f
			}

		case "status":
			if err := dec.Decode(&s.Status); err != nil {
				return fmt.Errorf("%s | %w", "Status", err)
			}

		case "task_max_waiting_in_queue":
			if err := dec.Decode(&s.TaskMaxWaitingInQueue); err != nil {
				return fmt.Errorf("%s | %w", "TaskMaxWaitingInQueue", err)
			}

		case "task_max_waiting_in_queue_millis":
			if err := dec.Decode(&s.TaskMaxWaitingInQueueMillis); err != nil {
				return fmt.Errorf("%s | %w", "TaskMaxWaitingInQueueMillis", err)
			}

		case "timed_out":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseBool(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "TimedOut", err)
				}
				s.TimedOut = value
			case bool:
				s.TimedOut = v
			}

		case "unassigned_primary_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "UnassignedPrimaryShards", err)
				}
				s.UnassignedPrimaryShards = value
			case float64:
				f := int(v)
				s.UnassignedPrimaryShards = f
			}

		case "unassigned_shards":

			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "UnassignedShards", err)
				}
				s.UnassignedShards = value
			case float64:
				f := int(v)
				s.UnassignedShards = f
			}

		}
	}
	return nil
}

// NewHealthResponseBody returns a HealthResponseBody.
func NewHealthResponseBody() *HealthResponseBody {
	r := &HealthResponseBody{
		Indices: make(map[string]IndexHealthStats, 0),
	}

	return r
}
