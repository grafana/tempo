// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type IsListArguments[K any] struct {
	Target ottl.Getter[K]
}

func NewIsListFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("IsList", &IsListArguments[K]{}, createIsListFunction[K])
}

func createIsListFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*IsListArguments[K])

	if !ok {
		return nil, errors.New("IsListFactory args must be of type *IsListArguments[K]")
	}

	return isList(args.Target), nil
}

func isList[K any](target ottl.Getter[K]) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		val, err := target.Get(ctx, tCtx)
		if err != nil {
			return false, err
		}

		switch valType := val.(type) {
		case pcommon.Value:
			return valType.Type() == pcommon.ValueTypeSlice, nil

		case pcommon.Slice, plog.LogRecordSlice, plog.ResourceLogsSlice, plog.ScopeLogsSlice, pmetric.ExemplarSlice, pmetric.ExponentialHistogramDataPointSlice, pmetric.HistogramDataPointSlice, pmetric.MetricSlice, pmetric.NumberDataPointSlice, pmetric.ResourceMetricsSlice, pmetric.ScopeMetricsSlice, pmetric.SummaryDataPointSlice, pmetric.SummaryDataPointValueAtQuantileSlice, ptrace.ResourceSpansSlice, ptrace.ScopeSpansSlice, ptrace.SpanEventSlice, ptrace.SpanLinkSlice, ptrace.SpanSlice, []string, []bool, []int64, []float64, [][]byte, []any:
			return true, nil
		}

		return false, nil
	}
}
