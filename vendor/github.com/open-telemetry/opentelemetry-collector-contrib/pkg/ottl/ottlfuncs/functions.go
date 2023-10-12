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
		NewConcatFactory[K](),
		NewConvertCaseFactory[K](),
		NewDurationFactory[K](),
		NewExtractPatternsFactory[K](),
		NewFnvFactory[K](),
		NewHoursFactory[K](),
		NewIntFactory[K](),
		NewIsMapFactory[K](),
		NewIsMatchFactory[K](),
		NewIsStringFactory[K](),
		NewLenFactory[K](),
		NewLogFactory[K](),
		NewMicrosecondsFactory[K](),
		NewMillisecondsFactory[K](),
		NewMinutesFactory[K](),
		NewNanosecondsFactory[K](),
		NewParseJSONFactory[K](),
		NewSecondsFactory[K](),
		NewSHA1Factory[K](),
		NewSHA256Factory[K](),
		NewSpanIDFactory[K](),
		NewSplitFactory[K](),
		NewSubstringFactory[K](),
		NewTimeFactory[K](),
		NewTraceIDFactory[K](),
		NewUnixMicroFactory[K](),
		NewUnixMilliFactory[K](),
		NewUnixNanoFactory[K](),
		NewUnixSecondsFactory[K](),
		NewUUIDFactory[K](),
	}
}
