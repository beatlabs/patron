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

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// GcsRepository type.
//
// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/snapshot/_types/SnapshotRepository.ts#L45-L48
type GcsRepository struct {
	Settings GcsRepositorySettings `json:"settings"`
	Type     string                `json:"type,omitempty"`
	Uuid     *string               `json:"uuid,omitempty"`
}

func (s *GcsRepository) UnmarshalJSON(data []byte) error {

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

		case "settings":
			if err := dec.Decode(&s.Settings); err != nil {
				return fmt.Errorf("%s | %w", "Settings", err)
			}

		case "type":
			if err := dec.Decode(&s.Type); err != nil {
				return fmt.Errorf("%s | %w", "Type", err)
			}

		case "uuid":
			if err := dec.Decode(&s.Uuid); err != nil {
				return fmt.Errorf("%s | %w", "Uuid", err)
			}

		}
	}
	return nil
}

// MarshalJSON override marshalling to include literal value
func (s GcsRepository) MarshalJSON() ([]byte, error) {
	type innerGcsRepository GcsRepository
	tmp := innerGcsRepository{
		Settings: s.Settings,
		Type:     s.Type,
		Uuid:     s.Uuid,
	}

	tmp.Type = "gcs"

	return json.Marshal(tmp)
}

// NewGcsRepository returns a GcsRepository.
func NewGcsRepository() *GcsRepository {
	r := &GcsRepository{}

	return r
}

// true

type GcsRepositoryVariant interface {
	GcsRepositoryCaster() *GcsRepository
}

func (s *GcsRepository) GcsRepositoryCaster() *GcsRepository {
	return s
}
