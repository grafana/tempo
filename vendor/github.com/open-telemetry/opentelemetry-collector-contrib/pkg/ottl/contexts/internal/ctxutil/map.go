// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func GetMapValue[K any](ctx context.Context, tCtx K, m pcommon.Map, keys []ottl.Key[K]) (any, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("cannot get map value without keys")
	}

	s, err := keys[0].String(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if s == nil {
		resString, err := FetchValueFromExpression[K, string](ctx, tCtx, keys[0])
		if err != nil {
			return nil, fmt.Errorf("unable to resolve a string index in map: %w", err)
		}
		s = resString
	}

	val, ok := m.Get(*s)
	if !ok {
		return nil, nil
	}

	return getIndexableValue[K](ctx, tCtx, val, keys[1:])
}

func SetMapValue[K any](ctx context.Context, tCtx K, m pcommon.Map, keys []ottl.Key[K], val any) error {
	if len(keys) == 0 {
		return fmt.Errorf("cannot set map value without key")
	}

	s, err := keys[0].String(ctx, tCtx)
	if err != nil {
		return err
	}
	if s == nil {
		resString, err := FetchValueFromExpression[K, string](ctx, tCtx, keys[0])
		if err != nil {
			return fmt.Errorf("unable to resolve a string index in map: %w", err)
		}
		s = resString
	}

	currentValue, ok := m.Get(*s)
	if !ok {
		currentValue = m.PutEmpty(*s)
	}
	return SetIndexableValue[K](ctx, tCtx, currentValue, val, keys[1:])
}

func FetchValueFromExpression[K any, T int64 | string](ctx context.Context, tCtx K, key ottl.Key[K]) (*T, error) {
	p, err := key.ExpressionGetter(ctx, tCtx)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("invalid key type")
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
