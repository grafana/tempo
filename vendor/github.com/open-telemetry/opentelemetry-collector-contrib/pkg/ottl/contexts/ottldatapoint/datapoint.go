// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottldatapoint // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
)

var _ internal.ResourceContext = TransformContext{}
var _ internal.InstrumentationScopeContext = TransformContext{}

type TransformContext struct {
	dataPoint            interface{}
	metric               pmetric.Metric
	metrics              pmetric.MetricSlice
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(dataPoint interface{}, metric pmetric.Metric, metrics pmetric.MetricSlice, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource) TransformContext {
	return TransformContext{
		dataPoint:            dataPoint,
		metric:               metric,
		metrics:              metrics,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
	}
}

func (tCtx TransformContext) GetDataPoint() interface{} {
	return tCtx.dataPoint
}

func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

func (tCtx TransformContext) GetMetric() pmetric.Metric {
	return tCtx.metric
}

func (tCtx TransformContext) GetMetrics() pmetric.MetricSlice {
	return tCtx.metrics
}

func (tCtx TransformContext) getCache() pcommon.Map {
	return tCtx.cache
}

func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
	p, err := ottl.NewParser[TransformContext](
		functions,
		parsePath,
		telemetrySettings,
		ottl.WithEnumParser[TransformContext](parseEnum),
	)
	if err != nil {
		return ottl.Parser[TransformContext]{}, err
	}
	for _, opt := range options {
		opt(&p)
	}
	return p, nil
}

type StatementsOption func(*ottl.Statements[TransformContext])

func WithErrorMode(errorMode ottl.ErrorMode) StatementsOption {
	return func(s *ottl.Statements[TransformContext]) {
		ottl.WithErrorMode[TransformContext](errorMode)(s)
	}
}

func NewStatements(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementsOption) ottl.Statements[TransformContext] {
	s := ottl.NewStatements(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

var symbolTable = map[ottl.EnumSymbol]ottl.Enum{
	"FLAG_NONE":              0,
	"FLAG_NO_RECORDED_VALUE": 1,
}

func init() {
	for k, v := range internal.MetricSymbolTable {
		symbolTable[k] = v
	}
}

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		if enum, ok := symbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, fmt.Errorf("enum symbol not provided")
}

func parsePath(val *ottl.Path) (ottl.GetSetter[TransformContext], error) {
	if val != nil && len(val.Fields) > 0 {
		return newPathGetSetter(val.Fields)
	}
	return nil, fmt.Errorf("bad path %v", val)
}

func newPathGetSetter(path []ottl.Field) (ottl.GetSetter[TransformContext], error) {
	switch path[0].Name {
	case "cache":
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessCache(), nil
		}
		return accessCacheKey(mapKey), nil
	case "resource":
		return internal.ResourcePathGetSetter[TransformContext](path[1:])
	case "instrumentation_scope":
		return internal.ScopePathGetSetter[TransformContext](path[1:])
	case "metric":
		return internal.MetricPathGetSetter[TransformContext](path[1:])
	case "attributes":
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessAttributes(), nil
		}
		return accessAttributesKey(mapKey), nil
	case "start_time_unix_nano":
		return accessStartTimeUnixNano(), nil
	case "time_unix_nano":
		return accessTimeUnixNano(), nil
	case "value_double":
		return accessDoubleValue(), nil
	case "value_int":
		return accessIntValue(), nil
	case "exemplars":
		return accessExemplars(), nil
	case "flags":
		return accessFlags(), nil
	case "count":
		return accessCount(), nil
	case "sum":
		return accessSum(), nil
	case "bucket_counts":
		return accessBucketCounts(), nil
	case "explicit_bounds":
		return accessExplicitBounds(), nil
	case "scale":
		return accessScale(), nil
	case "zero_count":
		return accessZeroCount(), nil
	case "positive":
		if len(path) == 1 {
			return accessPositive(), nil
		}
		switch path[1].Name {
		case "offset":
			return accessPositiveOffset(), nil
		case "bucket_counts":
			return accessPositiveBucketCounts(), nil
		}
	case "negative":
		if len(path) == 1 {
			return accessNegative(), nil
		}
		switch path[1].Name {
		case "offset":
			return accessNegativeOffset(), nil
		case "bucket_counts":
			return accessNegativeBucketCounts(), nil
		}
	case "quantile_values":
		return accessQuantileValues(), nil
	}
	return nil, fmt.Errorf("invalid path expression %v", path)
}

func accessCache() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.getCache(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if m, ok := val.(pcommon.Map); ok {
				m.CopyTo(tCtx.getCache())
			}
			return nil
		},
	}
}

func accessCacheKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return internal.GetMapValue(tCtx.getCache(), keys)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetMapValue(tCtx.getCache(), keys, val)
		},
	}
}

func accessAttributes() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return tCtx.GetDataPoint().(pmetric.NumberDataPoint).Attributes(), nil
			case pmetric.HistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Attributes(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Attributes(), nil
			case pmetric.SummaryDataPoint:
				return tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Attributes(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				if attrs, ok := val.(pcommon.Map); ok {
					attrs.CopyTo(tCtx.GetDataPoint().(pmetric.NumberDataPoint).Attributes())
				}
			case pmetric.HistogramDataPoint:
				if attrs, ok := val.(pcommon.Map); ok {
					attrs.CopyTo(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Attributes())
				}
			case pmetric.ExponentialHistogramDataPoint:
				if attrs, ok := val.(pcommon.Map); ok {
					attrs.CopyTo(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Attributes())
				}
			case pmetric.SummaryDataPoint:
				if attrs, ok := val.(pcommon.Map); ok {
					attrs.CopyTo(tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Attributes())
				}
			}
			return nil
		},
	}
}

func accessAttributesKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return internal.GetMapValue(tCtx.GetDataPoint().(pmetric.NumberDataPoint).Attributes(), keys)
			case pmetric.HistogramDataPoint:
				return internal.GetMapValue(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Attributes(), keys)
			case pmetric.ExponentialHistogramDataPoint:
				return internal.GetMapValue(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Attributes(), keys)
			case pmetric.SummaryDataPoint:
				return internal.GetMapValue(tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Attributes(), keys)
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return internal.SetMapValue(tCtx.GetDataPoint().(pmetric.NumberDataPoint).Attributes(), keys, val)
			case pmetric.HistogramDataPoint:
				return internal.SetMapValue(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Attributes(), keys, val)
			case pmetric.ExponentialHistogramDataPoint:
				return internal.SetMapValue(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Attributes(), keys, val)
			case pmetric.SummaryDataPoint:
				return internal.SetMapValue(tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Attributes(), keys, val)
			}
			return nil
		},
	}
}

func accessStartTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return tCtx.GetDataPoint().(pmetric.NumberDataPoint).StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.HistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.HistogramDataPoint).StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).StartTimestamp().AsTime().UnixNano(), nil
			case pmetric.SummaryDataPoint:
				return tCtx.GetDataPoint().(pmetric.SummaryDataPoint).StartTimestamp().AsTime().UnixNano(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newTime, ok := val.(int64); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					tCtx.GetDataPoint().(pmetric.NumberDataPoint).SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.HistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.HistogramDataPoint).SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.ExponentialHistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.SummaryDataPoint:
					tCtx.GetDataPoint().(pmetric.SummaryDataPoint).SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				}
			}
			return nil
		},
	}
}

func accessTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return tCtx.GetDataPoint().(pmetric.NumberDataPoint).Timestamp().AsTime().UnixNano(), nil
			case pmetric.HistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Timestamp().AsTime().UnixNano(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Timestamp().AsTime().UnixNano(), nil
			case pmetric.SummaryDataPoint:
				return tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Timestamp().AsTime().UnixNano(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newTime, ok := val.(int64); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					tCtx.GetDataPoint().(pmetric.NumberDataPoint).SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.HistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.HistogramDataPoint).SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.ExponentialHistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				case pmetric.SummaryDataPoint:
					tCtx.GetDataPoint().(pmetric.SummaryDataPoint).SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTime)))
				}
			}
			return nil
		},
	}
}

func accessDoubleValue() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
				return numberDataPoint.DoubleValue(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newDouble, ok := val.(float64); ok {
				if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
					numberDataPoint.SetDoubleValue(newDouble)
				}
			}
			return nil
		},
	}
}

func accessIntValue() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
				return numberDataPoint.IntValue(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newInt, ok := val.(int64); ok {
				if numberDataPoint, ok := tCtx.GetDataPoint().(pmetric.NumberDataPoint); ok {
					numberDataPoint.SetIntValue(newInt)
				}
			}
			return nil
		},
	}
}

func accessExemplars() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return tCtx.GetDataPoint().(pmetric.NumberDataPoint).Exemplars(), nil
			case pmetric.HistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Exemplars(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Exemplars(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newExemplars, ok := val.(pmetric.ExemplarSlice); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					newExemplars.CopyTo(tCtx.GetDataPoint().(pmetric.NumberDataPoint).Exemplars())
				case pmetric.HistogramDataPoint:
					newExemplars.CopyTo(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Exemplars())
				case pmetric.ExponentialHistogramDataPoint:
					newExemplars.CopyTo(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Exemplars())
				}
			}
			return nil
		},
	}
}

func accessFlags() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.NumberDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.NumberDataPoint).Flags()), nil
			case pmetric.HistogramDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Flags()), nil
			case pmetric.ExponentialHistogramDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Flags()), nil
			case pmetric.SummaryDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Flags()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newFlags, ok := val.(int64); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.NumberDataPoint:
					tCtx.GetDataPoint().(pmetric.NumberDataPoint).SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.HistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.HistogramDataPoint).SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.ExponentialHistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).SetFlags(pmetric.DataPointFlags(newFlags))
				case pmetric.SummaryDataPoint:
					tCtx.GetDataPoint().(pmetric.SummaryDataPoint).SetFlags(pmetric.DataPointFlags(newFlags))
				}
			}
			return nil
		},
	}
}

func accessCount() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.HistogramDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Count()), nil
			case pmetric.ExponentialHistogramDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Count()), nil
			case pmetric.SummaryDataPoint:
				return int64(tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Count()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newCount, ok := val.(int64); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.HistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.HistogramDataPoint).SetCount(uint64(newCount))
				case pmetric.ExponentialHistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).SetCount(uint64(newCount))
				case pmetric.SummaryDataPoint:
					tCtx.GetDataPoint().(pmetric.SummaryDataPoint).SetCount(uint64(newCount))
				}
			}
			return nil
		},
	}
}

func accessSum() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			switch tCtx.GetDataPoint().(type) {
			case pmetric.HistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.HistogramDataPoint).Sum(), nil
			case pmetric.ExponentialHistogramDataPoint:
				return tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).Sum(), nil
			case pmetric.SummaryDataPoint:
				return tCtx.GetDataPoint().(pmetric.SummaryDataPoint).Sum(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newSum, ok := val.(float64); ok {
				switch tCtx.GetDataPoint().(type) {
				case pmetric.HistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.HistogramDataPoint).SetSum(newSum)
				case pmetric.ExponentialHistogramDataPoint:
					tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint).SetSum(newSum)
				case pmetric.SummaryDataPoint:
					tCtx.GetDataPoint().(pmetric.SummaryDataPoint).SetSum(newSum)
				}
			}
			return nil
		},
	}
}

func accessExplicitBounds() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
				return histogramDataPoint.ExplicitBounds().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newExplicitBounds, ok := val.([]float64); ok {
				if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
					histogramDataPoint.ExplicitBounds().FromRaw(newExplicitBounds)
				}
			}
			return nil
		},
	}
}

func accessBucketCounts() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
				return histogramDataPoint.BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newBucketCount, ok := val.([]uint64); ok {
				if histogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.HistogramDataPoint); ok {
					histogramDataPoint.BucketCounts().FromRaw(newBucketCount)
				}
			}
			return nil
		},
	}
}

func accessScale() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Scale()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newScale, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.SetScale(int32(newScale))
				}
			}
			return nil
		},
	}
}

func accessZeroCount() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.ZeroCount()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newZeroCount, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.SetZeroCount(uint64(newZeroCount))
				}
			}
			return nil
		},
	}
}

func accessPositive() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Positive(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newPositive, ok := val.(pmetric.ExponentialHistogramDataPointBuckets); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					newPositive.CopyTo(expoHistogramDataPoint.Positive())
				}
			}
			return nil
		},
	}
}

func accessPositiveOffset() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Positive().Offset()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newPositiveOffset, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Positive().SetOffset(int32(newPositiveOffset))
				}
			}
			return nil
		},
	}
}

func accessPositiveBucketCounts() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Positive().BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newPositiveBucketCounts, ok := val.([]uint64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Positive().BucketCounts().FromRaw(newPositiveBucketCounts)
				}
			}
			return nil
		},
	}
}

func accessNegative() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Negative(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newNegative, ok := val.(pmetric.ExponentialHistogramDataPointBuckets); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					newNegative.CopyTo(expoHistogramDataPoint.Negative())
				}
			}
			return nil
		},
	}
}

func accessNegativeOffset() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return int64(expoHistogramDataPoint.Negative().Offset()), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newNegativeOffset, ok := val.(int64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Negative().SetOffset(int32(newNegativeOffset))
				}
			}
			return nil
		},
	}
}

func accessNegativeBucketCounts() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
				return expoHistogramDataPoint.Negative().BucketCounts().AsRaw(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newNegativeBucketCounts, ok := val.([]uint64); ok {
				if expoHistogramDataPoint, ok := tCtx.GetDataPoint().(pmetric.ExponentialHistogramDataPoint); ok {
					expoHistogramDataPoint.Negative().BucketCounts().FromRaw(newNegativeBucketCounts)
				}
			}
			return nil
		},
	}
}

func accessQuantileValues() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			if summaryDataPoint, ok := tCtx.GetDataPoint().(pmetric.SummaryDataPoint); ok {
				return summaryDataPoint.QuantileValues(), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newQuantileValues, ok := val.(pmetric.SummaryDataPointValueAtQuantileSlice); ok {
				if summaryDataPoint, ok := tCtx.GetDataPoint().(pmetric.SummaryDataPoint); ok {
					newQuantileValues.CopyTo(summaryDataPoint.QuantileValues())
				}
			}
			return nil
		},
	}
}
