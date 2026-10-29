// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/xottl/ottlfuncs"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

// ExperimentalConverters returns the converters that are excluded from OTTL's
// stability guarantees. Components merge these into their function set to make
// them available in OTTL statements.
func ExperimentalConverters[K any]() []ottl.Factory[K] {
	return []ottl.Factory[K]{
		NewAllFactory[K](),
		NewAnyFactory[K](),
		NewFilterFactory[K](),
		NewFindFactory[K](),
		NewMapEachFactory[K](),
		NewMapKeysFactory[K](),
		NewReduceFactory[K](),
		NewWhenFactory[K](),
	}
}

// WithExperimentalConverters adds the experimental converters to the given function map and returns it.
func WithExperimentalConverters[K any](funcs map[string]ottl.Factory[K]) map[string]ottl.Factory[K] {
	for _, f := range ExperimentalConverters[K]() {
		funcs[f.Name()] = f
	}
	return funcs
}
