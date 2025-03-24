// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxlog"

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxerror"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

func PathGetSetter[K Context](path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, ctxerror.New("nil", "nil", Name, DocRef)
	}
	switch path.Name() {
	case "time_unix_nano":
		return accessTimeUnixNano[K](), nil
	case "observed_time_unix_nano":
		return accessObservedTimeUnixNano[K](), nil
	case "time":
		return accessTime[K](), nil
	case "observed_time":
		return accessObservedTime[K](), nil
	case "severity_number":
		return accessSeverityNumber[K](), nil
	case "severity_text":
		return accessSeverityText[K](), nil
	case "body":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringBody[K](), nil
			}
			return nil, ctxerror.New(nextPath.Name(), nextPath.String(), Name, DocRef)
		}
		if path.Keys() == nil {
			return accessBody[K](), nil
		}
		return accessBodyKey(path.Keys()), nil
	case "attributes":
		if path.Keys() == nil {
			return accessAttributes[K](), nil
		}
		return accessAttributesKey(path.Keys()), nil
	case "dropped_attributes_count":
		return accessDroppedAttributesCount[K](), nil
	case "flags":
		return accessFlags[K](), nil
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
			return nil, ctxerror.New(nextPath.Name(), path.String(), Name, DocRef)
		}
		return accessSpanID[K](), nil
	default:
		return nil, ctxerror.New(path.Name(), path.String(), Name, DocRef)
	}
}

func accessTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessObservedTimeUnixNano[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().ObservedTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().Timestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetLogRecord().SetTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessObservedTime[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().ObservedTimestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetLogRecord().SetObservedTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessSeverityNumber[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetLogRecord().SeverityNumber()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetSeverityNumber(plog.SeverityNumber(i))
			}
			return nil
		},
	}
}

func accessSeverityText[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().SeverityText(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if s, ok := val.(string); ok {
				tCtx.GetLogRecord().SetSeverityText(s)
			}
			return nil
		},
	}
}

func accessBody[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return ottlcommon.GetValue(tCtx.GetLogRecord().Body()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetValue(tCtx.GetLogRecord().Body(), val)
		},
	}
}

func accessBodyKey[K Context](key []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return ctxutil.GetMapValue[K](ctx, tCtx, tCtx.GetLogRecord().Body().Map(), key)
			case pcommon.ValueTypeSlice:
				return ctxutil.GetSliceValue[K](ctx, tCtx, tCtx.GetLogRecord().Body().Slice(), key)
			default:
				return nil, fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return ctxutil.SetMapValue[K](ctx, tCtx, tCtx.GetLogRecord().Body().Map(), key, val)
			case pcommon.ValueTypeSlice:
				return ctxutil.SetSliceValue[K](ctx, tCtx, tCtx.GetLogRecord().Body().Slice(), key, val)
			default:
				return fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
	}
}

func accessStringBody[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().Body().AsString(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetLogRecord().Body().SetStr(str)
			}
			return nil
		},
	}
}

func accessAttributes[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			return ctxutil.SetMap(tCtx.GetLogRecord().Attributes(), val)
		},
	}
}

func accessAttributesKey[K Context](key []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return ctxutil.GetMapValue[K](ctx, tCtx, tCtx.GetLogRecord().Attributes(), key)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return ctxutil.SetMapValue[K](ctx, tCtx, tCtx.GetLogRecord().Attributes(), key, val)
		},
	}
}

func accessDroppedAttributesCount[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetLogRecord().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessFlags[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetLogRecord().Flags()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetFlags(plog.LogRecordFlags(i))
			}
			return nil
		},
	}
}

func accessTraceID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().TraceID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTraceID, ok := val.(pcommon.TraceID); ok {
				tCtx.GetLogRecord().SetTraceID(newTraceID)
			}
			return nil
		},
	}
}

func accessStringTraceID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetLogRecord().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				id, err := ctxcommon.ParseTraceID(str)
				if err != nil {
					return err
				}
				tCtx.GetLogRecord().SetTraceID(id)
			}
			return nil
		},
	}
}

func accessSpanID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetLogRecord().SpanID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetLogRecord().SetSpanID(newSpanID)
			}
			return nil
		},
	}
}

func accessStringSpanID[K Context]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetLogRecord().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				id, err := ctxcommon.ParseSpanID(str)
				if err != nil {
					return err
				}
				tCtx.GetLogRecord().SetSpanID(id)
			}
			return nil
		},
	}
}
