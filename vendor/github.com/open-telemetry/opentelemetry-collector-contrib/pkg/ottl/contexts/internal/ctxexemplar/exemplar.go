// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxexemplar // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxexemplar"

import (
	"context"
	"encoding/hex"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
	}
	switch path.Name() {
	case "time_unix_nano":
		return accessTimeUnixNano[K](), nil
	case "time":
		return accessTime[K](), nil
	case "double_value":
		return accessDoubleValue[K](), nil
	case "int_value":
		return accessIntValue[K](), nil
	case "trace_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringTraceID[K](), nil
			}
			return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
		}
		return accessTraceID[K](), nil
	case "span_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringSpanID[K](), nil
			}
			return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
		}
		return accessSpanID[K](), nil
	case "filtered_attributes":
		if path.Keys() == nil {
			return accessFilteredAttributes[K](), nil
		}
		return accessFilteredAttributesKey(path.Keys()), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newTimestamp, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, newTimestamp)))
			return nil
		},
	}
}

func accessTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().Timestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newTimestamp, err := ctxutil.ExpectType[time.Time](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetTimestamp(pcommon.NewTimestampFromTime(newTimestamp))
			return nil
		},
	}
}

func accessDoubleValue[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().DoubleValue(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newVal, err := ctxutil.ExpectType[float64](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetDoubleValue(newVal)
			return nil
		},
	}
}

func accessIntValue[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().IntValue(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newVal, err := ctxutil.ExpectType[int64](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetIntValue(newVal)
			return nil
		},
	}
}

func accessTraceID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().TraceID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newTraceID, err := ctxutil.ExpectType[pcommon.TraceID](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetTraceID(newTraceID)
			return nil
		},
	}
}

func accessStringTraceID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetExemplar().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			str, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			id, err := ctxcommon.ParseTraceID(str)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetTraceID(id)
			return nil
		},
	}
}

func accessSpanID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().SpanID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			newSpanID, err := ctxutil.ExpectType[pcommon.SpanID](val)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetSpanID(newSpanID)
			return nil
		},
	}
}

func accessStringSpanID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetExemplar().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			str, err := ctxutil.ExpectType[string](val)
			if err != nil {
				return err
			}
			id, err := ctxcommon.ParseSpanID(str)
			if err != nil {
				return err
			}
			tCtx.GetExemplar().SetSpanID(id)
			return nil
		},
	}
}

func accessFilteredAttributes[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetExemplar().FilteredAttributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetMap(tCtx.GetExemplar().FilteredAttributes(), val)
		},
	}
}

func accessFilteredAttributesKey[K Context](key []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return ctxutil.GetMapValue[K](ctx, tCtx, tCtx.GetExemplar().FilteredAttributes(), key)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return ctxutil.SetMapValue[K](ctx, tCtx, tCtx.GetExemplar().FilteredAttributes(), key, val)
		},
	}
}
