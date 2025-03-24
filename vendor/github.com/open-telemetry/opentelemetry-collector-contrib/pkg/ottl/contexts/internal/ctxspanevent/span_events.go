// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxspanevent // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxspanevent"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
	}
	switch path.Name() {
	case "time_unix_nano":
		return accessSpanEventTimeUnixNano[K](), nil
	case "time":
		return accessSpanEventTime[K](), nil
	case "name":
		return accessSpanEventName[K](), nil
	case "attributes":
		if path.Keys() == nil {
			return accessSpanEventAttributes[K](), nil
		}
		return accessSpanEventAttributesKey(path.Keys()), nil
	case "dropped_attributes_count":
		return accessSpanEventDroppedAttributeCount[K](), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessSpanEventTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpanEvent().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTimestamp, ok := val.(int64); ok {
				tCtx.GetSpanEvent().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTimestamp)))
			}
			return nil
		},
	}
}

func accessSpanEventTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpanEvent().Timestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTimestamp, ok := val.(time.Time); ok {
				tCtx.GetSpanEvent().SetTimestamp(pcommon.NewTimestampFromTime(newTimestamp))
			}
			return nil
		},
	}
}

func accessSpanEventName[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpanEvent().Name(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newName, ok := val.(string); ok {
				tCtx.GetSpanEvent().SetName(newName)
			}
			return nil
		},
	}
}

func accessSpanEventAttributes[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpanEvent().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetMap(tCtx.GetSpanEvent().Attributes(), val)
		},
	}
}

func accessSpanEventAttributesKey[K Context](key []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return ctxutil.GetMapValue[K](ctx, tCtx, tCtx.GetSpanEvent().Attributes(), key)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return ctxutil.SetMapValue[K](ctx, tCtx, tCtx.GetSpanEvent().Attributes(), key, val)
		},
	}
}

func accessSpanEventDroppedAttributeCount[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpanEvent().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newCount, ok := val.(int64); ok {
				tCtx.GetSpanEvent().SetDroppedAttributesCount(uint32(newCount))
			}
			return nil
		},
	}
}
