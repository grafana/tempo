// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

type ResourceContext interface {
	GetResource() pcommon.Resource
	GetResourceSchemaURLItem() SchemaURLItem
}

func ResourcePathGetSetter[K ResourceContext](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return accessResource[K](), nil
	}
	switch path.Name() {
	case "attributes":
		if path.Keys() == nil {
			return accessResourceAttributes[K](), nil
		}
		return accessResourceAttributesKey[K](path.Keys()), nil
	case "dropped_attributes_count":
		return accessResourceDroppedAttributesCount[K](), nil
	case "schema_url":
		return accessResourceSchemaURLItem[K](), nil
	default:
		return nil, FormatDefaultErrorMessage(path.Name(), path.String(), "Resource", ResourceContextRef)
	}
}

func accessResource[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetResource(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newRes, ok := val.(pcommon.Resource); ok {
				newRes.CopyTo(tCtx.GetResource())
			}
			return nil
		},
	}
}

func accessResourceAttributes[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetResource().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetResource().Attributes())
			}
			return nil
		},
	}
}

func accessResourceAttributesKey[K ResourceContext](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return GetMapValue[K](ctx, tCtx, tCtx.GetResource().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return SetMapValue[K](ctx, tCtx, tCtx.GetResource().Attributes(), keys, val)
		},
	}
}

func accessResourceDroppedAttributesCount[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetResource().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetResource().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessResourceSchemaURLItem[K ResourceContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetResourceSchemaURLItem().SchemaUrl(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if schemaURL, ok := val.(string); ok {
				tCtx.GetResourceSchemaURLItem().SetSchemaUrl(schemaURL)
			}
			return nil
		},
	}
}
