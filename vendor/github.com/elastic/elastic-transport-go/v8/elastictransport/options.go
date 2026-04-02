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

package elastictransport

import (
	"errors"
	"fmt"
	"strings"
)

// Options is a slice of [Option] values that supports validation, iteration,
// and human-readable formatting. Convert a plain []Option with Options(slice).
type Options []Option

// Validate applies every option to a throwaway [Config] and returns any
// validation errors. This lets callers verify options before constructing a
// client with [NewClient]. Errors from multiple options are joined.
//
// Note: Validate checks option-level constraints (e.g. non-negative retries,
// valid gzip levels). It does not perform the full client construction that
// [NewClient] does (e.g. TLS setup, CA parsing).
func (os Options) Validate() error {
	var cfg Config
	return os.applyTo(&cfg)
}

// Visit calls fn for each option in order.
func (os Options) Visit(fn func(Option)) {
	for _, o := range os {
		fn(o)
	}
}

// String returns a newline-separated list of options with sensitive values
// redacted. It implements [fmt.Stringer].
func (os Options) String() string {
	return os.Describe(false)
}

// Describe returns a newline-separated list of options. When showSensitive is
// true, secret values (API keys, passwords, tokens, certificates) are
// included; otherwise they are replaced with "****".
func (os Options) Describe(showSensitive bool) string {
	parts := make([]string, len(os))
	for i, o := range os {
		parts[i] = o.Describe(showSensitive)
	}
	return strings.Join(parts, "\n")
}

// applyTo applies every option to cfg, collecting all errors.
func (os Options) applyTo(cfg *Config) error {
	var errs []error
	for _, o := range os {
		if o.apply != nil {
			if err := o.apply(cfg); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("transport options: %w", errors.Join(errs...))
	}
	return nil
}
