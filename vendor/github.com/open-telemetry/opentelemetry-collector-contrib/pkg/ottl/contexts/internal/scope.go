// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type InstrumentationScopeContext interface {
	GetInstrumentationScope() pcommon.InstrumentationScope
	GetScopeSchemaURLItem() SchemaURLItem
}

func ScopePathGetSetter[K InstrumentationScopeContext](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return accessInstrumentationScope[K](), nil
	}
	switch path.Name() {
	case "name":
		return accessInstrumentationScopeName[K](), nil
	case "version":
		return accessInstrumentationScopeVersion[K](), nil
	case "attributes":
		mapKeys := path.Keys()
		if mapKeys == nil {
			return accessInstrumentationScopeAttributes[K](), nil
		}
		return accessInstrumentationScopeAttributesKey[K](mapKeys), nil
	case "dropped_attributes_count":
		return accessInstrumentationScopeDroppedAttributesCount[K](), nil
	case "schema_url":
		return accessInstrumentationScopeSchemaURLItem[K](), nil
	default:
		return nil, FormatDefaultErrorMessage(path.Name(), path.String(), "Instrumentation Scope", InstrumentationScopeRef)
	}
}

func accessInstrumentationScope[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetInstrumentationScope(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newIl, ok := val.(pcommon.InstrumentationScope); ok {
				newIl.CopyTo(tCtx.GetInstrumentationScope())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributes[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetInstrumentationScope().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetInstrumentationScope().Attributes())
			}
			return nil
		},
	}
}

func accessInstrumentationScopeAttributesKey[K InstrumentationScopeContext](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return GetMapValue[K](ctx, tCtx, tCtx.GetInstrumentationScope().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return SetMapValue[K](ctx, tCtx, tCtx.GetInstrumentationScope().Attributes(), keys, val)
		},
	}
}

func accessInstrumentationScopeName[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetInstrumentationScope().Name(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetName(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeVersion[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetInstrumentationScope().Version(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetInstrumentationScope().SetVersion(str)
			}
			return nil
		},
	}
}

func accessInstrumentationScopeDroppedAttributesCount[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetInstrumentationScope().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetInstrumentationScope().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessInstrumentationScopeSchemaURLItem[K InstrumentationScopeContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetScopeSchemaURLItem().SchemaUrl(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if schemaURL, ok := val.(string); ok {
				tCtx.GetScopeSchemaURLItem().SetSchemaUrl(schemaURL)
			}
			return nil
		},
	}
}
