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
)

func DefaultResourceFunctions() []ottl.Factory[ottlresource.TransformContext] {
	return slices.Collect(maps.Values(defaultResourceFunctionsMap()))
}

func DefaultLogFunctions() []ottl.Factory[ottllog.TransformContext] {
	return slices.Collect(maps.Values(defaultLogFunctionsMap()))
}

func DefaultMetricFunctions() []ottl.Factory[ottlmetric.TransformContext] {
	return slices.Collect(maps.Values(defaultMetricFunctionsMap()))
}

func DefaultDataPointFunctions() []ottl.Factory[ottldatapoint.TransformContext] {
	return slices.Collect(maps.Values(defaultDataPointFunctionsMap()))
}

func DefaultSpanFunctions() []ottl.Factory[ottlspan.TransformContext] {
	return slices.Collect(maps.Values(defaultSpanFunctionsMap()))
}

func DefaultSpanEventFunctions() []ottl.Factory[ottlspanevent.TransformContext] {
	return slices.Collect(maps.Values(defaultSpanEventFunctionsMap()))
}

func DefaultProfileFunctions() []ottl.Factory[ottlprofile.TransformContext] {
	return slices.Collect(maps.Values(defaultProfileFunctionsMap()))
}

func defaultResourceFunctionsMap() map[string]ottl.Factory[ottlresource.TransformContext] {
	return filterottl.StandardResourceFuncs()
}

func defaultLogFunctionsMap() map[string]ottl.Factory[ottllog.TransformContext] {
	return filterottl.StandardLogFuncs()
}

func defaultMetricFunctionsMap() map[string]ottl.Factory[ottlmetric.TransformContext] {
	return filterottl.StandardMetricFuncs()
}

func defaultDataPointFunctionsMap() map[string]ottl.Factory[ottldatapoint.TransformContext] {
	return filterottl.StandardDataPointFuncs()
}

func defaultSpanFunctionsMap() map[string]ottl.Factory[ottlspan.TransformContext] {
	return filterottl.StandardSpanFuncs()
}

func defaultSpanEventFunctionsMap() map[string]ottl.Factory[ottlspanevent.TransformContext] {
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
