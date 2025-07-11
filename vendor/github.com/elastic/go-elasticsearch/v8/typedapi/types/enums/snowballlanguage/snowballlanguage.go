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

// Package snowballlanguage
package snowballlanguage

import "strings"

// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/_types/analysis/languages.ts#L20-L48
type SnowballLanguage struct {
	Name string
}

var (
	Arabic = SnowballLanguage{"Arabic"}

	Armenian = SnowballLanguage{"Armenian"}

	Basque = SnowballLanguage{"Basque"}

	Catalan = SnowballLanguage{"Catalan"}

	Danish = SnowballLanguage{"Danish"}

	Dutch = SnowballLanguage{"Dutch"}

	English = SnowballLanguage{"English"}

	Estonian = SnowballLanguage{"Estonian"}

	Finnish = SnowballLanguage{"Finnish"}

	French = SnowballLanguage{"French"}

	German = SnowballLanguage{"German"}

	German2 = SnowballLanguage{"German2"}

	Hungarian = SnowballLanguage{"Hungarian"}

	Italian = SnowballLanguage{"Italian"}

	Irish = SnowballLanguage{"Irish"}

	Kp = SnowballLanguage{"Kp"}

	Lithuanian = SnowballLanguage{"Lithuanian"}

	Lovins = SnowballLanguage{"Lovins"}

	Norwegian = SnowballLanguage{"Norwegian"}

	Porter = SnowballLanguage{"Porter"}

	Portuguese = SnowballLanguage{"Portuguese"}

	Romanian = SnowballLanguage{"Romanian"}

	Russian = SnowballLanguage{"Russian"}

	Serbian = SnowballLanguage{"Serbian"}

	Spanish = SnowballLanguage{"Spanish"}

	Swedish = SnowballLanguage{"Swedish"}

	Turkish = SnowballLanguage{"Turkish"}
)

func (s SnowballLanguage) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

func (s *SnowballLanguage) UnmarshalText(text []byte) error {
	switch strings.ReplaceAll(strings.ToLower(string(text)), "\"", "") {

	case "Arabic":
		*s = Arabic
	case "Armenian":
		*s = Armenian
	case "Basque":
		*s = Basque
	case "Catalan":
		*s = Catalan
	case "Danish":
		*s = Danish
	case "Dutch":
		*s = Dutch
	case "English":
		*s = English
	case "Estonian":
		*s = Estonian
	case "Finnish":
		*s = Finnish
	case "French":
		*s = French
	case "German":
		*s = German
	case "German2":
		*s = German2
	case "Hungarian":
		*s = Hungarian
	case "Italian":
		*s = Italian
	case "Irish":
		*s = Irish
	case "Kp":
		*s = Kp
	case "Lithuanian":
		*s = Lithuanian
	case "Lovins":
		*s = Lovins
	case "Norwegian":
		*s = Norwegian
	case "Porter":
		*s = Porter
	case "Portuguese":
		*s = Portuguese
	case "Romanian":
		*s = Romanian
	case "Russian":
		*s = Russian
	case "Serbian":
		*s = Serbian
	case "Spanish":
		*s = Spanish
	case "Swedish":
		*s = Swedish
	case "Turkish":
		*s = Turkish
	default:
		*s = SnowballLanguage{string(text)}
	}

	return nil
}

func (s SnowballLanguage) String() string {
	return s.Name
}
