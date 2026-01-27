// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"maps"
	"slices"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

func DefaultResourceFunctions() []ottl.Factory[*ottlresource.TransformContext] {
	return slices.Collect(maps.Values(defaultResourceFunctionsMap()))
}

// Deprecated: [v0.142.0] use DefaultLogFunctionsNew.
func DefaultLogFunctions() []ottl.Factory[ottllog.TransformContext] {
	return slices.Collect(maps.Values(ottlfuncs.StandardConverters[ottllog.TransformContext]()))
}

func DefaultLogFunctionsNew() []ottl.Factory[*ottllog.TransformContext] {
	return slices.Collect(maps.Values(defaultLogFunctionsMap()))
}

// Deprecated: [v0.142.0] use DefaultMetricFunctionsNew.
func DefaultMetricFunctions() []ottl.Factory[ottlmetric.TransformContext] {
	m := ottlfuncs.StandardConverters[ottlmetric.TransformContext]()
	hasAttributeOnDatapointFactory := filterottl.NewHasAttributeOnDatapointFactory()
	hasAttributeKeyOnDatapointFactory := filterottl.NewHasAttributeKeyOnDatapointFactory()
	m[hasAttributeOnDatapointFactory.Name()] = hasAttributeOnDatapointFactory
	m[hasAttributeKeyOnDatapointFactory.Name()] = hasAttributeKeyOnDatapointFactory
	return slices.Collect(maps.Values(m))
}

func DefaultMetricFunctionsNew() []ottl.Factory[*ottlmetric.TransformContext] {
	return slices.Collect(maps.Values(defaultMetricFunctionsMap()))
}

// Deprecated: [v0.142.0] use DefaultDataPointFunctionsNew.
func DefaultDataPointFunctions() []ottl.Factory[ottldatapoint.TransformContext] {
	return slices.Collect(maps.Values(ottlfuncs.StandardConverters[ottldatapoint.TransformContext]()))
}

func DefaultDataPointFunctionsNew() []ottl.Factory[*ottldatapoint.TransformContext] {
	return slices.Collect(maps.Values(defaultDataPointFunctionsMap()))
}

// Deprecated: [v0.142.0] use DefaultSpanFunctionsNew.
func DefaultSpanFunctions() []ottl.Factory[ottlspan.TransformContext] {
	funcs := ottlfuncs.StandardConverters[ottlspan.TransformContext]()
	isRootSpanFactory := ottlfuncs.NewIsRootSpanFactory()
	funcs[isRootSpanFactory.Name()] = isRootSpanFactory
	return slices.Collect(maps.Values(funcs))
}

func DefaultSpanFunctionsNew() []ottl.Factory[*ottlspan.TransformContext] {
	return slices.Collect(maps.Values(defaultSpanFunctionsMap()))
}

// Deprecated: [v0.142.0] use DefaultSpanEventFunctionsNew.
func DefaultSpanEventFunctions() []ottl.Factory[*ottlspanevent.TransformContext] {
	return slices.Collect(maps.Values(ottlfuncs.StandardConverters[*ottlspanevent.TransformContext]()))
}

func DefaultSpanEventFunctionsNew() []ottl.Factory[*ottlspanevent.TransformContext] {
	return slices.Collect(maps.Values(defaultSpanEventFunctionsMap()))
}

func DefaultProfileFunctions() []ottl.Factory[ottlprofile.TransformContext] {
	return slices.Collect(maps.Values(defaultProfileFunctionsMap()))
}

func defaultResourceFunctionsMap() map[string]ottl.Factory[*ottlresource.TransformContext] {
	return filterottl.StandardResourceFuncs()
}

func defaultLogFunctionsMap() map[string]ottl.Factory[*ottllog.TransformContext] {
	return filterottl.StandardLogFuncs()
}

func defaultMetricFunctionsMap() map[string]ottl.Factory[*ottlmetric.TransformContext] {
	return filterottl.StandardMetricFuncs()
}

func defaultDataPointFunctionsMap() map[string]ottl.Factory[*ottldatapoint.TransformContext] {
	return filterottl.StandardDataPointFuncs()
}

func defaultSpanFunctionsMap() map[string]ottl.Factory[*ottlspan.TransformContext] {
	return filterottl.StandardSpanFuncs()
}

func defaultSpanEventFunctionsMap() map[string]ottl.Factory[*ottlspanevent.TransformContext] {
	return filterottl.StandardSpanEventFuncs()
}

func defaultProfileFunctionsMap() map[string]ottl.Factory[ottlprofile.TransformContext] {
	return filterottl.StandardProfileFuncs()
}

func mergeFunctionsToMap[K any](functionMap map[string]ottl.Factory[K], functions []ottl.Factory[K]) map[string]ottl.Factory[K] {
	for _, f := range functions {
		functionMap[f.Name()] = f
	}
	return functionMap
}
