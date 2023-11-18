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

package changepassword

import (
	"encoding/json"
	"fmt"
)

// Request holds the request body struct for the package changepassword
//
// https://github.com/elastic/elasticsearch-specification/blob/ac9c431ec04149d9048f2b8f9731e3c2f7f38754/specification/security/change_password/SecurityChangePasswordRequest.ts#L23-L51
type Request struct {

	// Password The new password value. Passwords must be at least 6 characters long.
	Password *string `json:"password,omitempty"`
	// PasswordHash A hash of the new password value. This must be produced using the same
	// hashing algorithm as has been configured for password storage. For more
	// details,
	// see the explanation of the `xpack.security.authc.password_hashing.algorithm`
	// setting.
	PasswordHash *string `json:"password_hash,omitempty"`
}

// NewRequest returns a Request
func NewRequest() *Request {
	r := &Request{}
	return r
}

// FromJSON allows to load an arbitrary json into the request structure
func (r *Request) FromJSON(data string) (*Request, error) {
	var req Request
	err := json.Unmarshal([]byte(data), &req)

	if err != nil {
		return nil, fmt.Errorf("could not deserialise json into Changepassword request: %w", err)
	}

	return &req, nil
}
