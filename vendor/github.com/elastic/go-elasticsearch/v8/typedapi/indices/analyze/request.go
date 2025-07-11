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

package analyze

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// Request holds the request body struct for the package analyze
//
// https://github.com/elastic/elasticsearch-specification/blob/3a94b6715915b1e9311724a2614c643368eece90/specification/indices/analyze/IndicesAnalyzeRequest.ts#L27-L119
type Request struct {

	// Analyzer The name of the analyzer that should be applied to the provided `text`.
	// This could be a built-in analyzer, or an analyzer that’s been configured in
	// the index.
	Analyzer *string `json:"analyzer,omitempty"`
	// Attributes Array of token attributes used to filter the output of the `explain`
	// parameter.
	Attributes []string `json:"attributes,omitempty"`
	// CharFilter Array of character filters used to preprocess characters before the
	// tokenizer.
	CharFilter []types.CharFilter `json:"char_filter,omitempty"`
	// Explain If `true`, the response includes token attributes and additional details.
	Explain *bool `json:"explain,omitempty"`
	// Field Field used to derive the analyzer.
	// To use this parameter, you must specify an index.
	// If specified, the `analyzer` parameter overrides this value.
	Field *string `json:"field,omitempty"`
	// Filter Array of token filters used to apply after the tokenizer.
	Filter []types.TokenFilter `json:"filter,omitempty"`
	// Normalizer Normalizer to use to convert text into a single token.
	Normalizer *string `json:"normalizer,omitempty"`
	// Text Text to analyze.
	// If an array of strings is provided, it is analyzed as a multi-value field.
	Text []string `json:"text,omitempty"`
	// Tokenizer Tokenizer to use to convert text into tokens.
	Tokenizer types.Tokenizer `json:"tokenizer,omitempty"`
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
		return nil, fmt.Errorf("could not deserialise json into Analyze request: %w", err)
	}

	return &req, nil
}

func (s *Request) UnmarshalJSON(data []byte) error {
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

		case "analyzer":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "Analyzer", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.Analyzer = &o

		case "attributes":
			if err := dec.Decode(&s.Attributes); err != nil {
				return fmt.Errorf("%s | %w", "Attributes", err)
			}

		case "char_filter":

			buf := []json.RawMessage{}
			dec.Decode(&buf)
			for _, rawMsg := range buf {

				source := bytes.NewReader(rawMsg)
				localDec := json.NewDecoder(source)
				kind := make(map[string]string, 0)
				localDec.Decode(&kind)
				source.Seek(0, io.SeekStart)

				switch kind["type"] {

				case "html_strip":
					o := types.NewHtmlStripCharFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				case "mapping":
					o := types.NewMappingCharFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				case "pattern_replace":
					o := types.NewPatternReplaceCharFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				case "icu_normalizer":
					o := types.NewIcuNormalizationCharFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				case "kuromoji_iteration_mark":
					o := types.NewKuromojiIterationMarkCharFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				default:
					o := new(any)
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.CharFilter = append(s.CharFilter, *o)
				}
			}

		case "explain":
			var tmp any
			dec.Decode(&tmp)
			switch v := tmp.(type) {
			case string:
				value, err := strconv.ParseBool(v)
				if err != nil {
					return fmt.Errorf("%s | %w", "Explain", err)
				}
				s.Explain = &value
			case bool:
				s.Explain = &v
			}

		case "field":
			if err := dec.Decode(&s.Field); err != nil {
				return fmt.Errorf("%s | %w", "Field", err)
			}

		case "filter":

			buf := []json.RawMessage{}
			dec.Decode(&buf)
			for _, rawMsg := range buf {

				source := bytes.NewReader(rawMsg)
				localDec := json.NewDecoder(source)
				kind := make(map[string]string, 0)
				localDec.Decode(&kind)
				source.Seek(0, io.SeekStart)

				switch kind["type"] {

				case "apostrophe":
					o := types.NewApostropheTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "arabic_normalization":
					o := types.NewArabicNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "asciifolding":
					o := types.NewAsciiFoldingTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "cjk_bigram":
					o := types.NewCjkBigramTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "cjk_width":
					o := types.NewCjkWidthTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "classic":
					o := types.NewClassicTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "common_grams":
					o := types.NewCommonGramsTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "condition":
					o := types.NewConditionTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "decimal_digit":
					o := types.NewDecimalDigitTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "delimited_payload":
					o := types.NewDelimitedPayloadTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "edge_ngram":
					o := types.NewEdgeNGramTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "elision":
					o := types.NewElisionTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "fingerprint":
					o := types.NewFingerprintTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "flatten_graph":
					o := types.NewFlattenGraphTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "german_normalization":
					o := types.NewGermanNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "hindi_normalization":
					o := types.NewHindiNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "hunspell":
					o := types.NewHunspellTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "hyphenation_decompounder":
					o := types.NewHyphenationDecompounderTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "indic_normalization":
					o := types.NewIndicNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "keep_types":
					o := types.NewKeepTypesTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "keep":
					o := types.NewKeepWordsTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "keyword_marker":
					o := types.NewKeywordMarkerTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "keyword_repeat":
					o := types.NewKeywordRepeatTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "kstem":
					o := types.NewKStemTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "length":
					o := types.NewLengthTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "limit":
					o := types.NewLimitTokenCountTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "lowercase":
					o := types.NewLowercaseTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "min_hash":
					o := types.NewMinHashTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "multiplexer":
					o := types.NewMultiplexerTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "ngram":
					o := types.NewNGramTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "nori_part_of_speech":
					o := types.NewNoriPartOfSpeechTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "pattern_capture":
					o := types.NewPatternCaptureTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "pattern_replace":
					o := types.NewPatternReplaceTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "persian_normalization":
					o := types.NewPersianNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "porter_stem":
					o := types.NewPorterStemTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "predicate_token_filter":
					o := types.NewPredicateTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "remove_duplicates":
					o := types.NewRemoveDuplicatesTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "reverse":
					o := types.NewReverseTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "scandinavian_folding":
					o := types.NewScandinavianFoldingTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "scandinavian_normalization":
					o := types.NewScandinavianNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "serbian_normalization":
					o := types.NewSerbianNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "shingle":
					o := types.NewShingleTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "snowball":
					o := types.NewSnowballTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "sorani_normalization":
					o := types.NewSoraniNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "stemmer_override":
					o := types.NewStemmerOverrideTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "stemmer":
					o := types.NewStemmerTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "stop":
					o := types.NewStopTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "synonym_graph":
					o := types.NewSynonymGraphTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "synonym":
					o := types.NewSynonymTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "trim":
					o := types.NewTrimTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "truncate":
					o := types.NewTruncateTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "unique":
					o := types.NewUniqueTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "uppercase":
					o := types.NewUppercaseTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "word_delimiter_graph":
					o := types.NewWordDelimiterGraphTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "word_delimiter":
					o := types.NewWordDelimiterTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "ja_stop":
					o := types.NewJaStopTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "kuromoji_stemmer":
					o := types.NewKuromojiStemmerTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "kuromoji_readingform":
					o := types.NewKuromojiReadingFormTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "kuromoji_part_of_speech":
					o := types.NewKuromojiPartOfSpeechTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "icu_collation":
					o := types.NewIcuCollationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "icu_folding":
					o := types.NewIcuFoldingTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "icu_normalizer":
					o := types.NewIcuNormalizationTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "icu_transform":
					o := types.NewIcuTransformTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "phonetic":
					o := types.NewPhoneticTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				case "dictionary_decompounder":
					o := types.NewDictionaryDecompounderTokenFilter()
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				default:
					o := new(any)
					if err := localDec.Decode(&o); err != nil {
						return err
					}
					s.Filter = append(s.Filter, *o)
				}
			}

		case "normalizer":
			var tmp json.RawMessage
			if err := dec.Decode(&tmp); err != nil {
				return fmt.Errorf("%s | %w", "Normalizer", err)
			}
			o := string(tmp[:])
			o, err = strconv.Unquote(o)
			if err != nil {
				o = string(tmp[:])
			}
			s.Normalizer = &o

		case "text":
			rawMsg := json.RawMessage{}
			dec.Decode(&rawMsg)
			if !bytes.HasPrefix(rawMsg, []byte("[")) {
				o := new(string)
				if err := json.NewDecoder(bytes.NewReader(rawMsg)).Decode(&o); err != nil {
					return fmt.Errorf("%s | %w", "Text", err)
				}

				s.Text = append(s.Text, *o)
			} else {
				if err := json.NewDecoder(bytes.NewReader(rawMsg)).Decode(&s.Text); err != nil {
					return fmt.Errorf("%s | %w", "Text", err)
				}
			}

		case "tokenizer":

			rawMsg := json.RawMessage{}
			dec.Decode(&rawMsg)
			source := bytes.NewReader(rawMsg)
			kind := make(map[string]string, 0)
			localDec := json.NewDecoder(source)
			localDec.Decode(&kind)
			source.Seek(0, io.SeekStart)

			switch kind["type"] {

			case "char_group":
				o := types.NewCharGroupTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "classic":
				o := types.NewClassicTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "edge_ngram":
				o := types.NewEdgeNGramTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "keyword":
				o := types.NewKeywordTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "letter":
				o := types.NewLetterTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "lowercase":
				o := types.NewLowercaseTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "ngram":
				o := types.NewNGramTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "path_hierarchy":
				o := types.NewPathHierarchyTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "pattern":
				o := types.NewPatternTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "simple_pattern":
				o := types.NewSimplePatternTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "simple_pattern_split":
				o := types.NewSimplePatternSplitTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "standard":
				o := types.NewStandardTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "thai":
				o := types.NewThaiTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "uax_url_email":
				o := types.NewUaxEmailUrlTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "whitespace":
				o := types.NewWhitespaceTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "icu_tokenizer":
				o := types.NewIcuTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "kuromoji_tokenizer":
				o := types.NewKuromojiTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			case "nori_tokenizer":
				o := types.NewNoriTokenizer()
				if err := localDec.Decode(&o); err != nil {
					return err
				}
				s.Tokenizer = *o
			default:
				if err := localDec.Decode(&s.Tokenizer); err != nil {
					return err
				}
			}

		}
	}
	return nil
}
