// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filtermetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermetric"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
)

// nameMatcher matches metrics by metric properties against prespecified values for each property.
type nameMatcher struct {
	nameFilters filterset.FilterSet
}

func newNameMatcher(mp *filterconfig.MetricMatchProperties) (*nameMatcher, error) {
	nameFS, err := filterset.CreateFilterSet(
		mp.MetricNames,
		&filterset.Config{
			MatchType:    filterset.MatchType(mp.MatchType),
			RegexpConfig: mp.RegexpConfig,
		},
	)
	if err != nil {
		return nil, err
	}
	return &nameMatcher{nameFilters: nameFS}, nil
}

// Eval matches a metric using the metric properties configured on the nameMatcher.
// A metric only matches if every metric property configured on the nameMatcher is a match.
func (m *nameMatcher) Eval(_ context.Context, tCtx ottlmetric.TransformContext) (bool, error) {
	return m.nameFilters.Matches(tCtx.GetMetric().Name()), nil
}
