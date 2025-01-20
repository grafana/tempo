// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type FlattenArguments[K any] struct {
	Target ottl.PMapGetter[K]
	Prefix ottl.Optional[string]
	Depth  ottl.Optional[int64]
}

func NewFlattenFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("flatten", &FlattenArguments[K]{}, createFlattenFunction[K])
}

func createFlattenFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FlattenArguments[K])

	if !ok {
		return nil, fmt.Errorf("FlattenFactory args must be of type *FlattenArguments[K]")
	}

	return flatten(args.Target, args.Prefix, args.Depth)
}

func flatten[K any](target ottl.PMapGetter[K], p ottl.Optional[string], d ottl.Optional[int64]) (ottl.ExprFunc[K], error) {
	depth := int64(math.MaxInt64)
	if !d.IsEmpty() {
		depth = d.Get()
		if depth < 1 {
			return nil, fmt.Errorf("invalid depth '%d' for flatten function, must be greater than 0", depth)
		}
	}

	var prefix string
	if !p.IsEmpty() {
		prefix = p.Get()
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		m, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		result := pcommon.NewMap()
		flattenMap(m, result, prefix, 0, depth)
		result.MoveTo(m)

		return nil, nil
	}, nil
}

func flattenMap(m pcommon.Map, result pcommon.Map, prefix string, currentDepth, maxDepth int64) {
	if len(prefix) > 0 {
		prefix += "."
	}
	m.Range(func(k string, v pcommon.Value) bool {
		return flattenValue(k, v, currentDepth, maxDepth, result, prefix)
	})
}

func flattenSlice(s pcommon.Slice, result pcommon.Map, prefix string, currentDepth int64, maxDepth int64) {
	for i := 0; i < s.Len(); i++ {
		flattenValue(fmt.Sprintf("%d", i), s.At(i), currentDepth+1, maxDepth, result, prefix)
	}
}

func flattenValue(k string, v pcommon.Value, currentDepth int64, maxDepth int64, result pcommon.Map, prefix string) bool {
	switch {
	case v.Type() == pcommon.ValueTypeMap && currentDepth < maxDepth:
		flattenMap(v.Map(), result, prefix+k, currentDepth+1, maxDepth)
	case v.Type() == pcommon.ValueTypeSlice && currentDepth < maxDepth:
		for i := 0; i < v.Slice().Len(); i++ {
			switch {
			case v.Slice().At(i).Type() == pcommon.ValueTypeMap && currentDepth+1 < maxDepth:
				flattenMap(v.Slice().At(i).Map(), result, fmt.Sprintf("%v.%v", prefix+k, i), currentDepth+2, maxDepth)
			case v.Slice().At(i).Type() == pcommon.ValueTypeSlice && currentDepth+1 < maxDepth:
				flattenSlice(v.Slice().At(i).Slice(), result, fmt.Sprintf("%v.%v", prefix+k, i), currentDepth+2, maxDepth)
			default:
				v.Slice().At(i).CopyTo(result.PutEmpty(fmt.Sprintf("%v.%v", prefix+k, i)))
			}
		}
	default:
		v.CopyTo(result.PutEmpty(prefix + k))
	}
	return true
}
