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

package getsynonymrule

// Response holds the response body struct for the package getsynonymrule
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/synonyms/get_synonym_rule/SynonymRuleGetResponse.ts#L22-L24
type Response struct {

	// Id Synonym Rule identifier
	Id string `json:"id"`
	// Synonyms Synonyms, in Solr format, that conform the synonym rule. See
	// https://www.elastic.co/guide/en/elasticsearch/reference/current/analysis-synonym-graph-tokenfilter.html#_solr_synonyms_2
	Synonyms string `json:"synonyms"`
}

// NewResponse returns a Response
func NewResponse() *Response {
	r := &Response{}
	return r
}
