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
// https://github.com/elastic/elasticsearch-specification/tree/4fcf747dfafc951e1dcf3077327e3dcee9107db3

// Package connectorstatus
package connectorstatus

import "strings"

// https://github.com/elastic/elasticsearch-specification/blob/4fcf747dfafc951e1dcf3077327e3dcee9107db3/specification/connector/_types/Connector.ts#L130-L136
type ConnectorStatus struct {
	Name string
}

var (
	Created = ConnectorStatus{"created"}

	Needsconfiguration = ConnectorStatus{"needs_configuration"}

	Configured = ConnectorStatus{"configured"}

	Connected = ConnectorStatus{"connected"}

	Error = ConnectorStatus{"error"}
)

func (c ConnectorStatus) MarshalText() (text []byte, err error) {
	return []byte(c.String()), nil
}

func (c *ConnectorStatus) UnmarshalText(text []byte) error {
	switch strings.ReplaceAll(strings.ToLower(string(text)), "\"", "") {

	case "created":
		*c = Created
	case "needs_configuration":
		*c = Needsconfiguration
	case "configured":
		*c = Configured
	case "connected":
		*c = Connected
	case "error":
		*c = Error
	default:
		*c = ConnectorStatus{string(text)}
	}

	return nil
}

func (c ConnectorStatus) String() string {
	return c.Name
}
