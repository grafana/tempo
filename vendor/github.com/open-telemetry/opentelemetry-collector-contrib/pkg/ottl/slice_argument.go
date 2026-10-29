// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import (
	"fmt"
	"reflect"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/slicegetter"
)

var errDynamicSliceArgumentsDisabled = fmt.Errorf(
	"must be a list; passing a path or converter that resolves to a slice requires the `%s` feature gate to be enabled",
	metadata.PkgOttlFunctionsEnableDynamicSliceArgumentsFeatureGate.ID(),
)

func isLiteralSliceElementType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Uint8, reflect.Float64, reflect.Int64:
		return true
	default:
		return false
	}
}

func (p *parseContext[K]) buildSliceGetterArg(fieldAddr any, arg argument, allowDynamic bool) (any, error) {
	sliceItemType, ok := slicegetter.ReflectTypeParam(fieldAddr)
	if !ok {
		return nil, ottlerror.New(arg.Pos, "slice getter type is not manageable by the OTTL parser. This is a bug in OTTL")
	}
	gv, err := buildSliceGetterValue[K](
		arg.Value,
		sliceItemType,
		allowDynamic,
		p.buildSliceArg,
		p.buildStandardGetSetter,
		p.newGetter,
	)
	if err != nil {
		return nil, ottlerror.FromError(arg.Pos, err)
	}
	if err := slicegetter.SetReflectValue(fieldAddr, reflect.ValueOf(gv)); err != nil {
		return nil, ottlerror.FromError(arg.Pos, err)
	}
	return reflect.ValueOf(fieldAddr).Elem().Interface(), nil
}

func buildSliceGetterValue[K any](
	val value,
	sliceItemType reflect.Type,
	allowDynamic bool,
	buildSliceArg func(value, reflect.Type) (any, error),
	buildSliceItemGetter func(string, Getter[K]) (any, error),
	buildGetter func(value) (Getter[K], error),
) (any, error) {
	if val.List != nil || isLiteralSliceElementType(sliceItemType) {
		return buildSliceArg(val, reflect.SliceOf(sliceItemType))
	}
	if !allowDynamic {
		return nil, errDynamicSliceArgumentsDisabled
	}

	valueGetter, err := buildGetter(val)
	if err != nil {
		return nil, err
	}

	return newRuntimeSliceSource(valueGetter, isLiteralGetter(valueGetter), sliceItemType, buildSliceItemGetter), nil
}

func newRuntimeSliceSource[K any](
	getter Getter[K],
	isLiteral bool,
	sliceItemType reflect.Type,
	buildSliceItemGetter func(string, Getter[K]) (any, error),
) any {
	return slicegetter.NewRuntimeSliceSource[K](
		getter,
		isLiteral,
		sliceItemType,
		func(name string, item slicegetter.Getter[K]) (any, error) {
			itemGetter, err := buildSliceItemGetter(name, item)
			if err != nil {
				return nil, err
			}
			if itemGetter == nil || !reflect.TypeOf(itemGetter).AssignableTo(sliceItemType) {
				return nil, TypeError(fmt.Sprintf("expected slice item of type %s, got %s", sliceItemType, reflect.TypeOf(itemGetter)))
			}
			return itemGetter, nil
		},
		func(item any) slicegetter.Getter[K] {
			return newLiteral[K, any](item)
		},
	)
}
