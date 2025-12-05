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

import "net/http"

// RoundTripFunc denotes the function signature of the elastictransport.Interface#Perform method
type RoundTripFunc func(*http.Request) (*http.Response, error)

// InterceptorFunc is a function that takes a RoundTripFunc, next, and returns another RoundTripFunc.
// This allows a chain of Interceptors to be created.
type InterceptorFunc func(next RoundTripFunc) RoundTripFunc

// mergeInterceptors creates a new InterceptorFunc by combining an array of InterceptorFunc
// This allows us to create and store a single interceptor instead of recreating the chain on each request.
func mergeInterceptors(interceptors []InterceptorFunc) InterceptorFunc {
	return func(next RoundTripFunc) RoundTripFunc {
		if len(interceptors) == 0 {
			return next
		}
		var fn = next
		for i := len(interceptors) - 1; i >= 0; i-- {
			if interceptors[i] == nil {
				continue
			}
			fn = interceptors[i](fn)
		}
		return fn
	}
}
