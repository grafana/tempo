// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type FlattenArguments[K any] struct {
	Target           ottl.PMapGetter[K]
	Prefix           ottl.Optional[string]
	Depth            ottl.Optional[int64]
	ResolveConflicts ottl.Optional[bool]
}

type flattenData struct {
	result          pcommon.Map
	existingKeys    map[string]int
	resolveConflict bool
	maxDepth        int64
}

func NewFlattenFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("flatten", &FlattenArguments[K]{}, createFlattenFunction[K])
}

func createFlattenFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*FlattenArguments[K])

	if !ok {
		return nil, fmt.Errorf("FlattenFactory args must be of type *FlattenArguments[K]")
	}

	return flatten(args.Target, args.Prefix, args.Depth, args.ResolveConflicts)
}

func flatten[K any](target ottl.PMapGetter[K], p ottl.Optional[string], d ottl.Optional[int64], c ottl.Optional[bool]) (ottl.ExprFunc[K], error) {
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

	resolveConflict := false
	if !c.IsEmpty() {
		resolveConflict = c.Get()
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		m, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, err
		}

		flattenData := initFlattenData(resolveConflict, depth)
		flattenData.flattenMap(m, prefix, 0)
		flattenData.result.MoveTo(m)

		return nil, nil
	}, nil
}

func initFlattenData(resolveConflict bool, maxDepth int64) *flattenData {
	return &flattenData{
		result:          pcommon.NewMap(),
		existingKeys:    map[string]int{},
		resolveConflict: resolveConflict,
		maxDepth:        maxDepth,
	}
}

func (f *flattenData) flattenMap(m pcommon.Map, prefix string, currentDepth int64) {
	if len(prefix) > 0 {
		prefix += "."
	}
	for k, v := range m.All() {
		f.flattenValue(k, v, currentDepth, prefix)
	}
}

func (f *flattenData) flattenSlice(s pcommon.Slice, prefix string, currentDepth int64) {
	for i := 0; i < s.Len(); i++ {
		f.flattenValue(fmt.Sprintf("%d", i), s.At(i), currentDepth+1, prefix)
	}
}

func (f *flattenData) flattenValue(k string, v pcommon.Value, currentDepth int64, prefix string) {
	switch {
	case v.Type() == pcommon.ValueTypeMap && currentDepth < f.maxDepth:
		f.flattenMap(v.Map(), prefix+k, currentDepth+1)
	case v.Type() == pcommon.ValueTypeSlice && currentDepth < f.maxDepth:
		for i := 0; i < v.Slice().Len(); i++ {
			switch {
			case v.Slice().At(i).Type() == pcommon.ValueTypeMap && currentDepth+1 < f.maxDepth:
				f.flattenMap(v.Slice().At(i).Map(), fmt.Sprintf("%v.%v", prefix+k, i), currentDepth+2)
			case v.Slice().At(i).Type() == pcommon.ValueTypeSlice && currentDepth+1 < f.maxDepth:
				f.flattenSlice(v.Slice().At(i).Slice(), fmt.Sprintf("%v.%v", prefix+k, i), currentDepth+2)
			default:
				key := prefix + k
				if f.resolveConflict {
					f.handleConflict(key, v.Slice().At(i))
				} else {
					v.Slice().At(i).CopyTo(f.result.PutEmpty(fmt.Sprintf("%v.%v", key, i)))
				}
			}
		}
	default:
		key := prefix + k
		if f.resolveConflict {
			f.handleConflict(key, v)
		} else {
			v.CopyTo(f.result.PutEmpty(key))
		}
	}
}

func (f *flattenData) handleConflict(key string, v pcommon.Value) {
	if _, exists := f.result.Get(key); exists {
		newKey := key + "." + strconv.Itoa(f.existingKeys[key])
		f.existingKeys[key]++
		v.CopyTo(f.result.PutEmpty(newKey))
	} else {
		f.existingKeys[key] = 0
		v.CopyTo(f.result.PutEmpty(key))
	}
}
