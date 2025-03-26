// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxscope // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
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
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessInstrumentationScopeAttributes[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetInstrumentationScope().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetMap(tCtx.GetInstrumentationScope().Attributes(), val)
		},
	}
}

func accessInstrumentationScopeAttributesKey[K Context](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return ctxutil.GetMapValue[K](ctx, tCtx, tCtx.GetInstrumentationScope().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return ctxutil.SetMapValue[K](ctx, tCtx, tCtx.GetInstrumentationScope().Attributes(), keys, val)
		},
	}
}

func accessInstrumentationScopeName[K Context]() ottl.StandardGetSetter[K] {
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

func accessInstrumentationScopeVersion[K Context]() ottl.StandardGetSetter[K] {
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

func accessInstrumentationScopeDroppedAttributesCount[K Context]() ottl.StandardGetSetter[K] {
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

func accessInstrumentationScopeSchemaURLItem[K Context]() ottl.StandardGetSetter[K] {
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
