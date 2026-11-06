// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/xprofile/ottlfuncs"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

// ProfileConverters returns the converters that operate on profiles data and are
// excluded from OTTL's stability guarantees. Components merge these into their
// function set to make them available in OTTL statements.
func ProfileConverters[K any]() []ottl.Factory[K] {
	return []ottl.Factory[K]{
		NewProfileIDFactory[K](),
	}
}

// WithProfileConverters adds the profiles converters to the given function map and returns it.
func WithProfileConverters[K any](funcs map[string]ottl.Factory[K]) map[string]ottl.Factory[K] {
	for _, f := range ProfileConverters[K]() {
		funcs[f.Name()] = f
	}
	return funcs
}
