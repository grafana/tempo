// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	typeError = `target arg must be of type string, []any, map[string]any, pcommon.Map, pcommon.Slice, pcommon.Value (of type String, Map, Slice) or a supported slice type from the plog, pmetric or ptrace packages`
)

type LenArguments[K any] struct {
	Target ottl.Getter[K]
}

func NewLenFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("Len", &LenArguments[K]{}, createLenFunction[K])
}

func createLenFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*LenArguments[K])

	if !ok {
		return nil, fmt.Errorf("LenFactory args must be of type *LenArguments[K]")
	}

	return computeLen(args.Target), nil
}

// nolint:exhaustive
func computeLen[K any](target ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		switch valType := val.(type) {
		case pcommon.Value:
			switch valType.Type() {
			case pcommon.ValueTypeStr:
				return int64(len(valType.Str())), nil
			case pcommon.ValueTypeSlice:
				return int64(valType.Slice().Len()), nil
			case pcommon.ValueTypeMap:
				return int64(valType.Map().Len()), nil
			}
			return nil, fmt.Errorf(typeError)
		case pcommon.Map:
			return int64(valType.Len()), nil
		case pcommon.Slice:
			return int64(valType.Len()), nil

		case plog.LogRecordSlice:
			return int64(valType.Len()), nil
		case plog.ResourceLogsSlice:
			return int64(valType.Len()), nil
		case plog.ScopeLogsSlice:
			return int64(valType.Len()), nil

		case pmetric.ExemplarSlice:
			return int64(valType.Len()), nil
		case pmetric.ExponentialHistogramDataPointSlice:
			return int64(valType.Len()), nil
		case pmetric.HistogramDataPointSlice:
			return int64(valType.Len()), nil
		case pmetric.MetricSlice:
			return int64(valType.Len()), nil
		case pmetric.NumberDataPointSlice:
			return int64(valType.Len()), nil
		case pmetric.ResourceMetricsSlice:
			return int64(valType.Len()), nil
		case pmetric.ScopeMetricsSlice:
			return int64(valType.Len()), nil
		case pmetric.SummaryDataPointSlice:
			return int64(valType.Len()), nil
		case pmetric.SummaryDataPointValueAtQuantileSlice:
			return int64(valType.Len()), nil

		case ptrace.ResourceSpansSlice:
			return int64(valType.Len()), nil
		case ptrace.ScopeSpansSlice:
			return int64(valType.Len()), nil
		case ptrace.SpanEventSlice:
			return int64(valType.Len()), nil
		case ptrace.SpanLinkSlice:
			return int64(valType.Len()), nil
		case ptrace.SpanSlice:
			return int64(valType.Len()), nil
		}

		v := reflect.ValueOf(val)
		switch v.Kind() {
		case reflect.String, reflect.Map, reflect.Slice:
			return int64(v.Len()), nil
		}

		return nil, fmt.Errorf(typeError)
	}
}
