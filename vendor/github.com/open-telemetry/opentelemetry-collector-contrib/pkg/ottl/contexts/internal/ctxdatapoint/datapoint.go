// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxdatapoint // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxdatapoint"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
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
	case "attributes":
		if path.Keys() == nil {
			return accessAttributes[K](), nil
		}
		return accessAttributesKey(path.Keys()), nil
	case "start_time_unix_nano":
		return accessStartTimeUnixNano[K](), nil
	case "time_unix_nano":
		return accessTimeUnixNano[K](), nil
	case "start_time":
		return accessStartTime[K](), nil
	case "time":
		return accessTime[K](), nil
	case "value_double":
		return accessDoubleValue[K](), nil
	case "value_int":
		return accessIntValue[K](), nil
	case "exemplars":
		return accessExemplars[K](), nil
	case "flags":
		return accessFlags[K](), nil
	case "count":
		return accessCount[K](), nil
	case "sum":
		return accessSum[K](), nil
	case "bucket_counts":
		return accessBucketCounts[K](), nil
	case "explicit_bounds":
		return accessExplicitBounds[K](), nil
	case "scale":
		return accessScale[K](), nil
	case "zero_count":
		return accessZeroCount[K](), nil
	case "positive":
		nextPath := path.Next()
		if nextPath != nil {
			switch nextPath.Name() {
			case "offset":
				return accessPositiveOffset[K](), nil
			case "bucket_counts":
				return accessPositiveBucketCounts[K](), nil
			default:
				return nil, ctxerror.New(nextPath.Name(), path.String(), Name, DocRef)
			}
		}
		return accessPositive[K](), nil
	case "negative":
		nextPath := path.Next()
		if nextPath != nil {
			switch nextPath.Name() {
			case "offset":
				return accessNegativeOffset[K](), nil
			case "bucket_counts":
				return accessNegativeBucketCounts[K](), nil
			default:
				return nil, ctxerror.New(nextPath.Name(), path.String(), Name, DocRef)
			}
		}
		return accessNegative[K](), nil
	case "quantile_values":
		return accessQuantileValues[K](), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessAttributes[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.Attributes(), nil
			case pmetric.HistogramDataPoint:
				return dp.Attributes(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.Attributes(), nil
			case pmetric.SummaryDataPoint:
				return dp.Attributes(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return ctxutil.SetMap(dp.Attributes(), val)
			case pmetric.HistogramDataPoint:
				return ctxutil.SetMap(dp.Attributes(), val)
			case pmetric.ExponentialHistogramDataPoint:
				return ctxutil.SetMap(dp.Attributes(), val)
			case pmetric.SummaryDataPoint:
				return ctxutil.SetMap(dp.Attributes(), val)
			}
			return nil
		},
	}
}

func accessAttributesKey[K Context](key []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return ctxutil.GetMapValue(ctx, tCtx, dp.Attributes(), key)
			case pmetric.HistogramDataPoint:
				return ctxutil.GetMapValue(ctx, tCtx, dp.Attributes(), key)
			case pmetric.ExponentialHistogramDataPoint:
				return ctxutil.GetMapValue(ctx, tCtx, dp.Attributes(), key)
			case pmetric.SummaryDataPoint:
				return ctxutil.GetMapValue(ctx, tCtx, dp.Attributes(), key)
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return ctxutil.SetMapValue(ctx, tCtx, dp.Attributes(), key, val)
			case pmetric.HistogramDataPoint:
				return ctxutil.SetMapValue(ctx, tCtx, dp.Attributes(), key, val)
			case pmetric.ExponentialHistogramDataPoint:
				return ctxutil.SetMapValue(ctx, tCtx, dp.Attributes(), key, val)
			case pmetric.SummaryDataPoint:
				return ctxutil.SetMapValue(ctx, tCtx, dp.Attributes(), key, val)
			}
			return nil
		},
	}
}

func accessStartTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.HistogramDataPoint:
				return dp.StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.SummaryDataPoint:
				return dp.StartTimestamp().AsTime().UnixNano(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTime, ok := val.(int64); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.HistogramDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.SummaryDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				}
			}
			return nil
		},
	}
}

func accessStartTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.StartTimestamp().AsTime(), nil
			case pmetric.HistogramDataPoint:
				return dp.StartTimestamp().AsTime(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.StartTimestamp().AsTime(), nil
			case pmetric.SummaryDataPoint:
				return dp.StartTimestamp().AsTime(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTime, ok := val.(time.Time); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.HistogramDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.SummaryDataPoint:
					dp.SetStartTimestamp(pcommon.NewTimestampFromTime(newTime))
				}
			}
			return nil
		},
	}
}

func accessTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.Timestamp().AsTime().UnixNano(), nil
			case pmetric.HistogramDataPoint:
				return dp.Timestamp().AsTime().UnixNano(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.Timestamp().AsTime().UnixNano(), nil
			case pmetric.SummaryDataPoint:
				return dp.Timestamp().AsTime().UnixNano(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTime, ok := val.(int64); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.HistogramDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.SummaryDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				}
			}
			return nil
		},
	}
}

func accessTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.Timestamp().AsTime(), nil
			case pmetric.HistogramDataPoint:
				return dp.Timestamp().AsTime(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.Timestamp().AsTime(), nil
			case pmetric.SummaryDataPoint:
				return dp.Timestamp().AsTime(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTime, ok := val.(time.Time); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.HistogramDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(newTime))
				case pmetric.SummaryDataPoint:
					dp.SetTimestamp(pcommon.NewTimestampFromTime(newTime))
				}
			}
			return nil
		},
	}
}

func accessDoubleValue[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
				return numberDataPoint.DoubleValue(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newDouble, ok := val.(float64); ok {
				if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
					numberDataPoint.SetDoubleValue(newDouble)
				}
			}
			return nil
		},
	}
}

func accessIntValue[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
				return numberDataPoint.IntValue(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newInt, ok := val.(int64); ok {
				if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
					numberDataPoint.SetIntValue(newInt)
				}
			}
			return nil
		},
	}
}

func accessExemplars[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return dp.Exemplars(), nil
			case pmetric.HistogramDataPoint:
				return dp.Exemplars(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.Exemplars(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newExemplars, ok := val.(pmetric.ExemplarSlice); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					newExemplars.CopyTo(dp.Exemplars())
				case pmetric.HistogramDataPoint:
					newExemplars.CopyTo(dp.Exemplars())
				case pmetric.ExponentialHistogramDataPoint:
					newExemplars.CopyTo(dp.Exemplars())
				}
			}
			return nil
		},
	}
}

func accessFlags[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return int64(dp.Flags()), nil
			case pmetric.HistogramDataPoint:
				return int64(dp.Flags()), nil
			case pmetric.ExponentialHistogramDataPoint:
				return int64(dp.Flags()), nil
			case pmetric.SummaryDataPoint:
				return int64(dp.Flags()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newFlags, ok := val.(int64); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					dp.SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.HistogramDataPoint:
					dp.SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.SummaryDataPoint:
					dp.SetFlags(pmetric.DataPointFlags(newFlags))
				}
			}
			return nil
		},
	}
}

func accessCount[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.HistogramDataPoint:
				return int64(dp.Count()), nil
			case pmetric.ExponentialHistogramDataPoint:
				return int64(dp.Count()), nil
			case pmetric.SummaryDataPoint:
				return int64(dp.Count()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newCount, ok := val.(int64); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.HistogramDataPoint:
					dp.SetCount(uint64(newCount))
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetCount(uint64(newCount))
				case pmetric.SummaryDataPoint:
					dp.SetCount(uint64(newCount))
				}
			}
			return nil
		},
	}
}

func accessSum[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			switch dp := tCtx.GetDataPoint().(type) {
			case pmetric.HistogramDataPoint:
				return dp.Sum(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return dp.Sum(), nil
			case pmetric.SummaryDataPoint:
				return dp.Sum(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newSum, ok := val.(float64); ok {
				switch dp := tCtx.GetDataPoint().(type) {
				case pmetric.HistogramDataPoint:
					dp.SetSum(newSum)
				case pmetric.ExponentialHistogramDataPoint:
					dp.SetSum(newSum)
				case pmetric.SummaryDataPoint:
					dp.SetSum(newSum)
				}
			}
			return nil
		},
	}
}

func accessExplicitBounds[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
				return histogramDataPoint.ExplicitBounds().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newExplicitBounds, ok := val.([]float64); ok {
				if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
					histogramDataPoint.ExplicitBounds().FromRaw(newExplicitBounds)
				}
			}
			return nil
		},
	}
}

func accessBucketCounts[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
				return histogramDataPoint.BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newBucketCount, ok := val.([]uint64); ok {
				if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
					histogramDataPoint.BucketCounts().FromRaw(newBucketCount)
				}
			}
			return nil
		},
	}
}

func accessScale[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Scale()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newScale, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.SetScale(int32(newScale))
				}
			}
			return nil
		},
	}
}

func accessZeroCount[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.ZeroCount()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newZeroCount, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.SetZeroCount(uint64(newZeroCount))
				}
			}
			return nil
		},
	}
}

func accessPositive[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Positive(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newPositive, ok := val.(pmetric.ExponentialHistogramDataPointBuckets); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					newPositive.CopyTo(expoHistogramDataPoint.Positive())
				}
			}
			return nil
		},
	}
}

func accessPositiveOffset[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Positive().Offset()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newPositiveOffset, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Positive().SetOffset(int32(newPositiveOffset))
				}
			}
			return nil
		},
	}
}

func accessPositiveBucketCounts[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Positive().BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newPositiveBucketCounts, ok := val.([]uint64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Positive().BucketCounts().FromRaw(newPositiveBucketCounts)
				}
			}
			return nil
		},
	}
}

func accessNegative[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Negative(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newNegative, ok := val.(pmetric.ExponentialHistogramDataPointBuckets); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					newNegative.CopyTo(expoHistogramDataPoint.Negative())
				}
			}
			return nil
		},
	}
}

func accessNegativeOffset[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Negative().Offset()), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newNegativeOffset, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Negative().SetOffset(int32(newNegativeOffset))
				}
			}
			return nil
		},
	}
}

func accessNegativeBucketCounts[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Negative().BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newNegativeBucketCounts, ok := val.([]uint64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Negative().BucketCounts().FromRaw(newNegativeBucketCounts)
				}
			}
			return nil
		},
	}
}

func accessQuantileValues[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			if summaryDataPoint, ok := tCtx.GetDataPoint().(pmetric.SummaryDataPoint); ok {
				return summaryDataPoint.QuantileValues(), nil
			}
			return nil, nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newQuantileValues, ok := val.(pmetric.SummaryDataPointValueAtQuantileSlice); ok {
				if summaryDataPoint, ok := tCtx.GetDataPoint().(pmetric.SummaryDataPoint); ok {
					newQuantileValues.CopyTo(summaryDataPoint.QuantileValues())
				}
			}
			return nil
		},
	}
}
