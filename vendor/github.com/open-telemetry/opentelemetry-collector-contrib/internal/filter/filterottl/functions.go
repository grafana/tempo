// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

func StandardSpanFuncs() map[string]ottl.Factory[ottlspan.TransformContext] {
	return standardFuncs[ottlspan.TransformContext]()
}

func StandardSpanEventFuncs() map[string]ottl.Factory[ottlspanevent.TransformContext] {
	return standardFuncs[ottlspanevent.TransformContext]()
}

func StandardMetricFuncs() map[string]ottl.Factory[ottlmetric.TransformContext] {
	m := standardFuncs[ottlmetric.TransformContext]()
	hasAttributeOnDatapointFactory := newHasAttributeOnDatapointFactory()
	hasAttributeKeyOnDatapointFactory := newHasAttributeKeyOnDatapointFactory()
	m[hasAttributeOnDatapointFactory.Name()] = hasAttributeOnDatapointFactory
	m[hasAttributeKeyOnDatapointFactory.Name()] = hasAttributeKeyOnDatapointFactory
	return m
}

func StandardDataPointFuncs() map[string]ottl.Factory[ottldatapoint.TransformContext] {
	return standardFuncs[ottldatapoint.TransformContext]()
}

func StandardLogFuncs() map[string]ottl.Factory[ottllog.TransformContext] {
	return standardFuncs[ottllog.TransformContext]()
}

func StandardResourceFuncs() map[string]ottl.Factory[ottlresource.TransformContext] {
	return standardFuncs[ottlresource.TransformContext]()
}

func standardFuncs[K any]() map[string]ottl.Factory[K] {
	m := ottlfuncs.StandardConverters[K]()
	f := newDropFactory[K]()
	m[f.Name()] = f
	return m
}

func newDropFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("drop", nil, createDropFunction[K])
}

func createDropFunction[K any](_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[K], error) {
	return dropFn[K]()
}

func dropFn[K any]() (ottl.ExprFunc[K], error) {
	return func(context.Context, K) (interface{}, error) {
		return true, nil
	}, nil
}

type hasAttributeOnDatapointArguments struct {
	Key         string `ottlarg:"0"`
	ExpectedVal string `ottlarg:"1"`
}

func newHasAttributeOnDatapointFactory() ottl.Factory[ottlmetric.TransformContext] {
	return ottl.NewFactory("HasAttrOnDatapoint", &hasAttributeOnDatapointArguments{}, createHasAttributeOnDatapointFunction)
}

func createHasAttributeOnDatapointFunction(_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	args, ok := oArgs.(*hasAttributeOnDatapointArguments)

	if !ok {
		return nil, fmt.Errorf("hasAttributeOnDatapointFactory args must be of type *hasAttributeOnDatapointArguments")
	}

	return hasAttributeOnDatapoint(args.Key, args.ExpectedVal)
}

func hasAttributeOnDatapoint(key string, expectedVal string) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	return func(ctx context.Context, tCtx ottlmetric.TransformContext) (interface{}, error) {
		return checkDataPoints(tCtx, key, &expectedVal)
	}, nil
}

type hasAttributeKeyOnDatapointArguments struct {
	Key string `ottlarg:"0"`
}

func newHasAttributeKeyOnDatapointFactory() ottl.Factory[ottlmetric.TransformContext] {
	return ottl.NewFactory("HasAttrKeyOnDatapoint", &hasAttributeKeyOnDatapointArguments{}, createHasAttributeKeyOnDatapointFunction)
}

func createHasAttributeKeyOnDatapointFunction(_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	args, ok := oArgs.(*hasAttributeKeyOnDatapointArguments)

	if !ok {
		return nil, fmt.Errorf("hasAttributeKeyOnDatapointFactory args must be of type *hasAttributeOnDatapointArguments")
	}

	return hasAttributeKeyOnDatapoint(args.Key)
}

func hasAttributeKeyOnDatapoint(key string) (ottl.ExprFunc[ottlmetric.TransformContext], error) {
	return func(ctx context.Context, tCtx ottlmetric.TransformContext) (interface{}, error) {
		return checkDataPoints(tCtx, key, nil)
	}, nil
}

func checkDataPoints(tCtx ottlmetric.TransformContext, key string, expectedVal *string) (interface{}, error) {
	metric := tCtx.GetMetric()
	//exhaustive:enforce
	switch metric.Type() {
	case pmetric.MetricTypeSum:
		return checkNumberDataPointSlice(metric.Sum().DataPoints(), key, expectedVal), nil
	case pmetric.MetricTypeGauge:
		return checkNumberDataPointSlice(metric.Gauge().DataPoints(), key, expectedVal), nil
	case pmetric.MetricTypeHistogram:
		return checkHistogramDataPointSlice(metric.Histogram().DataPoints(), key, expectedVal), nil
	case pmetric.MetricTypeExponentialHistogram:
		return checkExponentialHistogramDataPointSlice(metric.ExponentialHistogram().DataPoints(), key, expectedVal), nil
	case pmetric.MetricTypeSummary:
		return checkSummaryDataPointSlice(metric.Summary().DataPoints(), key, expectedVal), nil
	}
	return nil, fmt.Errorf("unknown metric type")
}

func checkNumberDataPointSlice(dps pmetric.NumberDataPointSlice, key string, expectedVal *string) bool {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		value, ok := dp.Attributes().Get(key)
		if ok {
			if expectedVal != nil {
				return value.Str() == *expectedVal
			}
			return true
		}
	}
	return false
}

func checkHistogramDataPointSlice(dps pmetric.HistogramDataPointSlice, key string, expectedVal *string) bool {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		value, ok := dp.Attributes().Get(key)
		if ok {
			if expectedVal != nil {
				return value.Str() == *expectedVal
			}
			return true
		}
	}
	return false
}

func checkExponentialHistogramDataPointSlice(dps pmetric.ExponentialHistogramDataPointSlice, key string, expectedVal *string) bool {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		value, ok := dp.Attributes().Get(key)
		if ok {
			if expectedVal != nil {
				return value.Str() == *expectedVal
			}
			return true
		}
	}
	return false
}

func checkSummaryDataPointSlice(dps pmetric.SummaryDataPointSlice, key string, expectedVal *string) bool {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		value, ok := dp.Attributes().Get(key)
		if ok {
			if expectedVal != nil {
				return value.Str() == *expectedVal
			}
			return true
		}
	}
	return false
}
