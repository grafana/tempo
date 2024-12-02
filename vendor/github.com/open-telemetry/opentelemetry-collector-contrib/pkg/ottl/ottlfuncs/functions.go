// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func StandardFuncs[K any]() map[string]ottl.Factory[K] {
	f := []ottl.Factory[K]{
		// Editors
		NewDeleteKeyFactory[K](),
		NewDeleteMatchingKeysFactory[K](),
		NewKeepMatchingKeysFactory[K](),
		NewFlattenFactory[K](),
		NewKeepKeysFactory[K](),
		NewLimitFactory[K](),
		NewMergeMapsFactory[K](),
		NewReplaceAllMatchesFactory[K](),
		NewReplaceAllPatternsFactory[K](),
		NewReplaceMatchFactory[K](),
		NewReplacePatternFactory[K](),
		NewSetFactory[K](),
		NewTruncateAllFactory[K](),
	}
	f = append(f, converters[K]()...)

	return ottl.CreateFactoryMap(f...)
}

func StandardConverters[K any]() map[string]ottl.Factory[K] {
	return ottl.CreateFactoryMap(converters[K]()...)
}

func converters[K any]() []ottl.Factory[K] {
	return []ottl.Factory[K]{
		// Converters
		NewBase64DecodeFactory[K](),
		NewDecodeFactory[K](),
		NewConcatFactory[K](),
		NewConvertCaseFactory[K](),
		NewConvertAttributesToElementsXMLFactory[K](),
		NewConvertTextToElementsXMLFactory[K](),
		NewDayFactory[K](),
		NewDoubleFactory[K](),
		NewDurationFactory[K](),
		NewExtractPatternsFactory[K](),
		NewExtractGrokPatternsFactory[K](),
		NewFnvFactory[K](),
		NewGetXMLFactory[K](),
		NewHourFactory[K](),
		NewHoursFactory[K](),
		NewInsertXMLFactory[K](),
		NewIntFactory[K](),
		NewIsBoolFactory[K](),
		NewIsDoubleFactory[K](),
		NewIsListFactory[K](),
		NewIsIntFactory[K](),
		NewIsMapFactory[K](),
		NewIsMatchFactory[K](),
		NewIsStringFactory[K](),
		NewLenFactory[K](),
		NewLogFactory[K](),
		NewMD5Factory[K](),
		NewMicrosecondsFactory[K](),
		NewMillisecondsFactory[K](),
		NewMinuteFactory[K](),
		NewMinutesFactory[K](),
		NewMonthFactory[K](),
		NewNanosecondsFactory[K](),
		NewNowFactory[K](),
		NewParseCSVFactory[K](),
		NewParseJSONFactory[K](),
		NewParseKeyValueFactory[K](),
		NewParseSimplifiedXMLFactory[K](),
		NewParseXMLFactory[K](),
		NewRemoveXMLFactory[K](),
		NewSecondsFactory[K](),
		NewSHA1Factory[K](),
		NewSHA256Factory[K](),
		NewSHA512Factory[K](),
		NewSortFactory[K](),
		NewSpanIDFactory[K](),
		NewSplitFactory[K](),
		NewFormatFactory[K](),
		NewStringFactory[K](),
		NewSubstringFactory[K](),
		NewTimeFactory[K](),
		NewToKeyValueStringFactory[K](),
		NewTruncateTimeFactory[K](),
		NewTraceIDFactory[K](),
		NewUnixFactory[K](),
		NewUnixMicroFactory[K](),
		NewUnixMilliFactory[K](),
		NewUnixNanoFactory[K](),
		NewUnixSecondsFactory[K](),
		NewUUIDFactory[K](),
		NewURLFactory[K](),
		NewUserAgentFactory[K](),
		NewAppendFactory[K](),
		NewYearFactory[K](),
		NewHexFactory[K](),
		NewSliceToMapFactory[K](),
	}
}
