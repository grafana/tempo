// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type InstrumentationScopeContext interface {
	GetInstrumentationScope() pcommon.InstrumentationScope
}

func ScopePathGetSetter[K InstrumentationScopeContext](path []ottl.Field) (ottl.GetSetter[K], error) {
	if len(path) == 0 {
		return accessInstrumentationScope[K](), nil
	}

	switch path[0].Name {
	case "name":
		return accessInstrumentationScopeName[K](), nil
	case "version":
		return accessInstrumentationScopeVersion[K](), nil
	case "attributes":
		mapKeys := path[0].Keys
		if mapKeys == nil {
			return accessInstrumentationScopeAttributes[K](), nil
		}
		return accessInstrumentationScopeAttributesKey[K](mapKeys), nil
	case "dropped_attributes_count":
		return accessInstrumentationScopeDroppedAttributesCount[K](), nil
	}

	return nil, fmt.Errorf("invalid scope path expression %v", path)
}

func accessInstrumentationScope[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newIl, ok := val.(pcommon.InstrumentationScope); ok {
				newIl.CopyTo(tCtx.GetInstrumentationScope())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributes[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetInstrumentationScope().Attributes())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributesKey[K InstrumentationScopeContext](keys []ottl.Key) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return GetMapValue(tCtx.GetInstrumentationScope().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			return SetMapValue(tCtx.GetInstrumentationScope().Attributes(), keys, val)
		},
	}
}

func accessInstrumentationScopeName[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Name(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetName(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeVersion[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetInstrumentationScope().Version(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetVersion(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeDroppedAttributesCount[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetInstrumentationScope().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetInstrumentationScope().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}
