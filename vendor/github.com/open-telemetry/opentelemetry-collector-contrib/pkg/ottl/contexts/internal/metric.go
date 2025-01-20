// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	MetricContextName = "metric"
)

type MetricContext interface {
	GetMetric() pmetric.Metric
}

var MetricSymbolTable = map[ottl.EnumSymbol]ottl.Enum{
	"AGGREGATION_TEMPORALITY_UNSPECIFIED":    ottl.Enum(pmetric.AggregationTemporalityUnspecified),
	"AGGREGATION_TEMPORALITY_DELTA":          ottl.Enum(pmetric.AggregationTemporalityDelta),
	"AGGREGATION_TEMPORALITY_CUMULATIVE":     ottl.Enum(pmetric.AggregationTemporalityCumulative),
	"METRIC_DATA_TYPE_NONE":                  ottl.Enum(pmetric.MetricTypeEmpty),
	"METRIC_DATA_TYPE_GAUGE":                 ottl.Enum(pmetric.MetricTypeGauge),
	"METRIC_DATA_TYPE_SUM":                   ottl.Enum(pmetric.MetricTypeSum),
	"METRIC_DATA_TYPE_HISTOGRAM":             ottl.Enum(pmetric.MetricTypeHistogram),
	"METRIC_DATA_TYPE_EXPONENTIAL_HISTOGRAM": ottl.Enum(pmetric.MetricTypeExponentialHistogram),
	"METRIC_DATA_TYPE_SUMMARY":               ottl.Enum(pmetric.MetricTypeSummary),
}

func MetricPathGetSetter[K MetricContext](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, FormatDefaultErrorMessage(MetricContextName, MetricContextName, "Metric", MetricRef)
	}
	switch path.Name() {
	case "name":
		return accessName[K](), nil
	case "description":
		return accessDescription[K](), nil
	case "unit":
		return accessUnit[K](), nil
	case "type":
		return accessType[K](), nil
	case "aggregation_temporality":
		return accessAggTemporality[K](), nil
	case "is_monotonic":
		return accessIsMonotonic[K](), nil
	case "data_points":
		return accessDataPoints[K](), nil
	default:
		return nil, FormatDefaultErrorMessage(path.Name(), path.String(), "Metric", MetricRef)
	}
}

func accessName[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Name(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetMetric().SetName(str)
			}
			return nil
		},
	}
}

func accessDescription[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Description(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetMetric().SetDescription(str)
			}
			return nil
		},
	}
}

func accessUnit[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Unit(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetMetric().SetUnit(str)
			}
			return nil
		},
	}
}

func accessType[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetMetric().Type()), nil
		},
		Setter: func(_ context.Context, _ K, _ any) error {
			// TODO Implement methods so correctly convert data types.
			// https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/10130
			return nil
		},
	}
}

func accessAggTemporality[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			metric := tCtx.GetMetric()
			switch metric.Type() {
			case pmetric.MetricTypeSum:
				return int64(metric.Sum().AggregationTemporality()), nil
			case pmetric.MetricTypeHistogram:
				return int64(metric.Histogram().AggregationTemporality()), nil
			case pmetric.MetricTypeExponentialHistogram:
				return int64(metric.ExponentialHistogram().AggregationTemporality()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newAggTemporality, ok := val.(int64); ok {
				metric := tCtx.GetMetric()
				switch metric.Type() {
				case pmetric.MetricTypeSum:
					metric.Sum().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
				case pmetric.MetricTypeHistogram:
					metric.Histogram().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
				case pmetric.MetricTypeExponentialHistogram:
					metric.ExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
				}
			}
			return nil
		},
	}
}

func accessIsMonotonic[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			metric := tCtx.GetMetric()
			if metric.Type() == pmetric.MetricTypeSum {
				return metric.Sum().IsMonotonic(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newIsMonotonic, ok := val.(bool); ok {
				metric := tCtx.GetMetric()
				if metric.Type() == pmetric.MetricTypeSum {
					metric.Sum().SetIsMonotonic(newIsMonotonic)
				}
			}
			return nil
		},
	}
}

func accessDataPoints[K MetricContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			metric := tCtx.GetMetric()
			switch metric.Type() {
			case pmetric.MetricTypeSum:
				return metric.Sum().DataPoints(), nil
			case pmetric.MetricTypeGauge:
				return metric.Gauge().DataPoints(), nil
			case pmetric.MetricTypeHistogram:
				return metric.Histogram().DataPoints(), nil
			case pmetric.MetricTypeExponentialHistogram:
				return metric.ExponentialHistogram().DataPoints(), nil
			case pmetric.MetricTypeSummary:
				return metric.Summary().DataPoints(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			metric := tCtx.GetMetric()
			switch metric.Type() {
			case pmetric.MetricTypeSum:
				if newDataPoints, ok := val.(pmetric.NumberDataPointSlice); ok {
					newDataPoints.CopyTo(metric.Sum().DataPoints())
				}
			case pmetric.MetricTypeGauge:
				if newDataPoints, ok := val.(pmetric.NumberDataPointSlice); ok {
					newDataPoints.CopyTo(metric.Gauge().DataPoints())
				}
			case pmetric.MetricTypeHistogram:
				if newDataPoints, ok := val.(pmetric.HistogramDataPointSlice); ok {
					newDataPoints.CopyTo(metric.Histogram().DataPoints())
				}
			case pmetric.MetricTypeExponentialHistogram:
				if newDataPoints, ok := val.(pmetric.ExponentialHistogramDataPointSlice); ok {
					newDataPoints.CopyTo(metric.ExponentialHistogram().DataPoints())
				}
			case pmetric.MetricTypeSummary:
				if newDataPoints, ok := val.(pmetric.SummaryDataPointSlice); ok {
					newDataPoints.CopyTo(metric.Summary().DataPoints())
				}
			}
			return nil
		},
	}
}
