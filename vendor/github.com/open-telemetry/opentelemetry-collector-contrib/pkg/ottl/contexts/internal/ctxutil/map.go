// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func GetMapValue[K any](ctx context.Context, tCtx K, m pcommon.Map, keys []ottl.Key[K]) (any, error) {
	if len(keys) == 0 {
		return nil, errors.New("cannot get map value without keys")
	}

	s, err := GetMapKeyName(ctx, tCtx, keys[0])
	if err != nil {
		return nil, fmt.Errorf("cannot get map value: %w", err)
	}

	val, ok := m.Get(*s)
	if !ok {
		return nil, nil
	}

	return getIndexableValue[K](ctx, tCtx, val, keys[1:])
}

func SetMapValue[K any](ctx context.Context, tCtx K, m pcommon.Map, keys []ottl.Key[K], val any) error {
	if len(keys) == 0 {
		return errors.New("cannot set map value without keys")
	}

	s, err := GetMapKeyName(ctx, tCtx, keys[0])
	if err != nil {
		return fmt.Errorf("cannot set map value: %w", err)
	}

	currentValue, ok := m.Get(*s)
	if !ok {
		currentValue = m.PutEmpty(*s)
	}
	return SetIndexableValue[K](ctx, tCtx, currentValue, val, keys[1:])
}

func GetMapKeyName[K any](ctx context.Context, tCtx K, key ottl.Key[K]) (*string, error) {
	resolvedKey, err := key.String(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if resolvedKey == nil {
		resolvedKey, err = FetchValueFromExpression[K, string](ctx, tCtx, key)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve a string index in map: %w", err)
		}
	}
	return resolvedKey, nil
}

func FetchValueFromExpression[K any, T int64 | string](ctx context.Context, tCtx K, key ottl.Key[K]) (*T, error) {
	p, err := key.ExpressionGetter(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("invalid key type")
	}
	res, err := p.Get(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	resVal, ok := res.(T)
	if !ok {
		return nil, fmt.Errorf("could not resolve key for map/slice, expecting '%T' but got '%T'", resVal, res)
	}
	return &resVal, nil
}

func SetMap(target pcommon.Map, val any) error {
	if cm, ok := val.(pcommon.Map); ok {
		cm.CopyTo(target)
		return nil
	}
	if rm, ok := val.(map[string]any); ok {
		return target.FromRaw(rm)
	}
	return nil
}

func GetMap(val any) (pcommon.Map, error) {
	if m, ok := val.(pcommon.Map); ok {
		return m, nil
	}
	if rm, ok := val.(map[string]any); ok {
		m := pcommon.NewMap()
		if err := m.FromRaw(rm); err != nil {
			return pcommon.Map{}, err
		}
		return m, nil
	}
	return pcommon.Map{}, fmt.Errorf("failed to convert type %T into pcommon.Map", val)
}
