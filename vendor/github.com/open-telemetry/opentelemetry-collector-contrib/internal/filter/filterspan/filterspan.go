// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterspan // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterspan"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filtermatcher"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
)

var useOTTLBridge = featuregate.GlobalRegistry().MustRegister(
	"filter.filterspan.useOTTLBridge",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("When enabled, filterspan will convert filterspan configuration to OTTL and use filterottl evaluation"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/18642"),
)

// NewSkipExpr creates a BoolExpr that on evaluation returns true if a span should NOT be processed or kept.
// The logic determining if a span should be processed is based on include and exclude settings.
// Include properties are checked before exclude settings are checked.
func NewSkipExpr(mp *filterconfig.MatchConfig) (expr.BoolExpr[ottlspan.TransformContext], error) {
	if useOTTLBridge.IsEnabled() {
		return filterottl.NewSpanSkipExprBridge(mp)
	}
	var matchers []expr.BoolExpr[ottlspan.TransformContext]
	inclExpr, err := newExpr(mp.Include)
	if err != nil {
		return nil, err
	}
	if inclExpr != nil {
		matchers = append(matchers, expr.Not(inclExpr))
	}
	exclExpr, err := newExpr(mp.Exclude)
	if err != nil {
		return nil, err
	}
	if exclExpr != nil {
		matchers = append(matchers, exclExpr)
	}
	return expr.Or(matchers...), nil
}

// propertiesMatcher allows matching a span against various span properties.
type propertiesMatcher struct {
	filtermatcher.PropertiesMatcher

	// Service names to compare to.
	serviceFilters filterset.FilterSet

	// Span names to compare to.
	nameFilters filterset.FilterSet

	// Span kinds to compare to
	kindFilters filterset.FilterSet
}

// newExpr creates a BoolExpr that matches based on the given MatchProperties.
func newExpr(mp *filterconfig.MatchProperties) (expr.BoolExpr[ottlspan.TransformContext], error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForSpans(); err != nil {
		return nil, err
	}

	rm, err := filtermatcher.NewMatcher(mp)
	if err != nil {
		return nil, err
	}

	var serviceFS filterset.FilterSet
	if len(mp.Services) > 0 {
		serviceFS, err = filterset.CreateFilterSet(mp.Services, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating service name filters: %w", err)
		}
	}

	var nameFS filterset.FilterSet
	if len(mp.SpanNames) > 0 {
		nameFS, err = filterset.CreateFilterSet(mp.SpanNames, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating span name filters: %w", err)
		}
	}

	var kindFS filterset.FilterSet
	if len(mp.SpanKinds) > 0 {
		kindFS, err = filterset.CreateFilterSet(mp.SpanKinds, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating span kind filters: %w", err)
		}
	}

	return &propertiesMatcher{
		PropertiesMatcher: rm,
		serviceFilters:    serviceFS,
		nameFilters:       nameFS,
		kindFilters:       kindFS,
	}, nil
}

// Eval matches a span and service to a set of properties.
// see filterconfig.MatchProperties for more details
func (mp *propertiesMatcher) Eval(_ context.Context, tCtx ottlspan.TransformContext) (bool, error) {
	// If a set of properties was not in the mp, all spans are considered to match on that property
	if mp.serviceFilters != nil {
		// Check resource and spans for service.name
		serviceName := serviceNameForResource(tCtx.GetResource())

		if !mp.serviceFilters.Matches(serviceName) {
			return false, nil
		}
	}

	if mp.nameFilters != nil && !mp.nameFilters.Matches(tCtx.GetSpan().Name()) {
		return false, nil
	}

	if mp.kindFilters != nil && !mp.kindFilters.Matches(traceutil.SpanKindStr(tCtx.GetSpan().Kind())) {
		return false, nil
	}

	return mp.PropertiesMatcher.Match(tCtx.GetSpan().Attributes(), tCtx.GetResource(), tCtx.GetInstrumentationScope()), nil
}

// serviceNameForResource gets the service name for a specified Resource.
func serviceNameForResource(resource pcommon.Resource) string {
	service, found := resource.Attributes().Get(conventions.AttributeServiceName)
	if !found {
		return "<nil-service-name>"
	}
	return service.AsString()
}
