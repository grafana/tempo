// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxmetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
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
	case "metadata":
		if path.Keys() == nil {
			return accessMetadata[K](), nil
		}
		return accessMetadataKey[K](path.Keys()), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessName[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Name(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			str, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			tCtx.GetMetric().SetName(str)
			return nil
		},
	}
}

func accessDescription[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Description(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			str, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			tCtx.GetMetric().SetDescription(str)
			return nil
		},
	}
}

func accessUnit[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Unit(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			str, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			tCtx.GetMetric().SetUnit(str)
			return nil
		},
	}
}

func accessType[K Context]() ottl.StandardGetSetter[K] {
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

func accessAggTemporality[K Context]() ottl.StandardGetSetter[K] {
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
			newAggTemporality, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			metric := tCtx.GetMetric()
			switch metric.Type() {
			case pmetric.MetricTypeSum:
				metric.Sum().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
			case pmetric.MetricTypeHistogram:
				metric.Histogram().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
			case pmetric.MetricTypeExponentialHistogram:
				metric.ExponentialHistogram().SetAggregationTemporality(pmetric.AggregationTemporality(newAggTemporality))
			}
			return nil
		},
	}
}

func accessIsMonotonic[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			metric := tCtx.GetMetric()
			if metric.Type() == pmetric.MetricTypeSum {
				return metric.Sum().IsMonotonic(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newIsMonotonic, err := ctxutil.ExpectType[bool](val)
			if err != nil {
				return err
			}
			metric := tCtx.GetMetric()
			if metric.Type() == pmetric.MetricTypeSum {
				metric.Sum().SetIsMonotonic(newIsMonotonic)
			}
			return nil
		},
	}
}

func accessDataPoints[K Context]() ottl.StandardGetSetter[K] {
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
				newDataPoints, err := ctxutil.ExpectType[pmetric.NumberDataPointSlice](val)
				if err != nil {
					return err
				}
				newDataPoints.CopyTo(metric.Sum().DataPoints())
			case pmetric.MetricTypeGauge:
				newDataPoints, err := ctxutil.ExpectType[pmetric.NumberDataPointSlice](val)
				if err != nil {
					return err
				}
				newDataPoints.CopyTo(metric.Gauge().DataPoints())
			case pmetric.MetricTypeHistogram:
				newDataPoints, err := ctxutil.ExpectType[pmetric.HistogramDataPointSlice](val)
				if err != nil {
					return err
				}
				newDataPoints.CopyTo(metric.Histogram().DataPoints())
			case pmetric.MetricTypeExponentialHistogram:
				newDataPoints, err := ctxutil.ExpectType[pmetric.ExponentialHistogramDataPointSlice](val)
				if err != nil {
					return err
				}
				newDataPoints.CopyTo(metric.ExponentialHistogram().DataPoints())
			case pmetric.MetricTypeSummary:
				newDataPoints, err := ctxutil.ExpectType[pmetric.SummaryDataPointSlice](val)
				if err != nil {
					return err
				}
				newDataPoints.CopyTo(metric.Summary().DataPoints())
			}
			return nil
		},
	}
}

func accessMetadata[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetMetric().Metadata(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetMap(tCtx.GetMetric().Metadata(), val)
		},
	}
}

func accessMetadataKey[K Context](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return ctxutil.GetMapValue(ctx, tCtx, tCtx.GetMetric().Metadata(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return ctxutil.SetMapValue(ctx, tCtx, tCtx.GetMetric().Metadata(), keys, val)
		},
	}
}
