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

// TokenFilterDefinition holds the union for the following types:
//
//	AsciiFoldingTokenFilter
//	CommonGramsTokenFilter
//	ConditionTokenFilter
//	DelimitedPayloadTokenFilter
//	EdgeNGramTokenFilter
//	ElisionTokenFilter
//	FingerprintTokenFilter
//	HunspellTokenFilter
//	HyphenationDecompounderTokenFilter
//	KeepTypesTokenFilter
//	KeepWordsTokenFilter
//	KeywordMarkerTokenFilter
//	KStemTokenFilter
//	LengthTokenFilter
//	LimitTokenCountTokenFilter
//	LowercaseTokenFilter
//	MultiplexerTokenFilter
//	NGramTokenFilter
//	NoriPartOfSpeechTokenFilter
//	PatternCaptureTokenFilter
//	PatternReplaceTokenFilter
//	PorterStemTokenFilter
//	PredicateTokenFilter
//	RemoveDuplicatesTokenFilter
//	ReverseTokenFilter
//	ShingleTokenFilter
//	SnowballTokenFilter
//	StemmerOverrideTokenFilter
//	StemmerTokenFilter
//	StopTokenFilter
//	SynonymGraphTokenFilter
//	SynonymTokenFilter
//	TrimTokenFilter
//	TruncateTokenFilter
//	UniqueTokenFilter
//	UppercaseTokenFilter
//	WordDelimiterGraphTokenFilter
//	WordDelimiterTokenFilter
//	KuromojiStemmerTokenFilter
//	KuromojiReadingFormTokenFilter
//	KuromojiPartOfSpeechTokenFilter
//	IcuCollationTokenFilter
//	IcuFoldingTokenFilter
//	IcuNormalizationTokenFilter
//	IcuTransformTokenFilter
//	PhoneticTokenFilter
//	DictionaryDecompounderTokenFilter
//
// https://github.com/elastic/elasticsearch-specification/blob/f6a370d0fba975752c644fc730f7c45610e28f36/specification/_types/analysis/token_filters.ts#L352-L404
type TokenFilterDefinition any

type TokenFilterDefinitionVariant interface {
	TokenFilterDefinitionCaster() *TokenFilterDefinition
}
