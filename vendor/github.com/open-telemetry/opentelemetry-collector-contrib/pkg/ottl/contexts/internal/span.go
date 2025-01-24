// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/trace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	SpanContextName            = "span"
	SpanContextNameDescription = "Span"
)

type SpanContext interface {
	GetSpan() ptrace.Span
}

var SpanSymbolTable = map[ottl.EnumSymbol]ottl.Enum{
	"SPAN_KIND_UNSPECIFIED": ottl.Enum(ptrace.SpanKindUnspecified),
	"SPAN_KIND_INTERNAL":    ottl.Enum(ptrace.SpanKindInternal),
	"SPAN_KIND_SERVER":      ottl.Enum(ptrace.SpanKindServer),
	"SPAN_KIND_CLIENT":      ottl.Enum(ptrace.SpanKindClient),
	"SPAN_KIND_PRODUCER":    ottl.Enum(ptrace.SpanKindProducer),
	"SPAN_KIND_CONSUMER":    ottl.Enum(ptrace.SpanKindConsumer),
	"STATUS_CODE_UNSET":     ottl.Enum(ptrace.StatusCodeUnset),
	"STATUS_CODE_OK":        ottl.Enum(ptrace.StatusCodeOk),
	"STATUS_CODE_ERROR":     ottl.Enum(ptrace.StatusCodeError),
}

func SpanPathGetSetter[K SpanContext](lowerContext string, path ottl.Path[K]) (ottl.GetSetter[K], error) {
	if path == nil {
		return nil, FormatDefaultErrorMessage(SpanContextName, SpanContextName, SpanContextNameDescription, SpanRef)
	}
	switch path.Name() {
	case "trace_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringTraceID[K](), nil
			}
			return nil, FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), SpanContextNameDescription, SpanRef)
		}
		return accessTraceID[K](), nil
	case "span_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringSpanID[K](), nil
			}
			return nil, FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), SpanContextNameDescription, SpanRef)
		}
		return accessSpanID[K](), nil
	case "trace_state":
		mapKey := path.Keys()
		if mapKey == nil {
			return accessTraceState[K](), nil
		}
		return accessTraceStateKey[K](mapKey)
	case "parent_span_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringParentSpanID[K](), nil
			}
			return nil, FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), SpanContextNameDescription, SpanRef)
		}
		return accessParentSpanID[K](), nil
	case "name":
		return accessSpanName[K](), nil
	case "kind":
		nextPath := path.Next()
		if nextPath != nil {
			switch nextPath.Name() {
			case "string":
				return accessStringKind[K](), nil
			case "deprecated_string":
				return accessDeprecatedStringKind[K](), nil
			default:
				return nil, FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), SpanContextNameDescription, SpanRef)
			}
		}
		return accessKind[K](), nil
	case "start_time_unix_nano":
		return accessStartTimeUnixNano[K](), nil
	case "end_time_unix_nano":
		return accessEndTimeUnixNano[K](), nil
	case "start_time":
		return accessStartTime[K](), nil
	case "end_time":
		return accessEndTime[K](), nil
	case "attributes":
		mapKeys := path.Keys()
		if mapKeys == nil {
			return accessAttributes[K](), nil
		}
		return accessAttributesKey[K](mapKeys), nil
	case "dropped_attributes_count":
		return accessSpanDroppedAttributesCount[K](), nil
	case "events":
		return accessEvents[K](), nil
	case "dropped_events_count":
		return accessDroppedEventsCount[K](), nil
	case "links":
		return accessLinks[K](), nil
	case "dropped_links_count":
		return accessDroppedLinksCount[K](), nil
	case "status":
		nextPath := path.Next()
		if nextPath != nil {
			switch nextPath.Name() {
			case "code":
				return accessStatusCode[K](), nil
			case "message":
				return accessStatusMessage[K](), nil
			default:
				return nil, FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), SpanContextNameDescription, SpanRef)
			}
		}
		return accessStatus[K](), nil
	case "cache":
		return nil, FormatCacheErrorMessage(lowerContext, path.Context(), path.String())
	default:
		return nil, FormatDefaultErrorMessage(path.Name(), path.String(), SpanContextNameDescription, SpanRef)
	}
}

func accessTraceID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().TraceID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newTraceID, ok := val.(pcommon.TraceID); ok {
				tCtx.GetSpan().SetTraceID(newTraceID)
			}
			return nil
		},
	}
}

func accessStringTraceID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetSpan().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				id, err := ParseTraceID(str)
				if err != nil {
					return err
				}
				tCtx.GetSpan().SetTraceID(id)
			}
			return nil
		},
	}
}

func accessSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().SpanID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetSpan().SetSpanID(newSpanID)
			}
			return nil
		},
	}
}

func accessStringSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetSpan().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				id, err := ParseSpanID(str)
				if err != nil {
					return err
				}
				tCtx.GetSpan().SetSpanID(id)
			}
			return nil
		},
	}
}

func accessTraceState[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().TraceState().AsRaw(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().TraceState().FromRaw(str)
			}
			return nil
		},
	}
}

func accessTraceStateKey[K SpanContext](keys []ottl.Key[K]) (ottl.StandardGetSetter[K], error) {
	if len(keys) != 1 {
		return ottl.StandardGetSetter[K]{}, fmt.Errorf("must provide exactly 1 key when accessing trace_state")
	}
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			if ts, err := trace.ParseTraceState(tCtx.GetSpan().TraceState().AsRaw()); err == nil {
				s, err := keys[0].String(ctx, tCtx)
				if err != nil {
					return nil, err
				}
				if s == nil {
					return nil, fmt.Errorf("trace_state indexing type must be a string")
				}
				return ts.Get(*s), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				if ts, err := trace.ParseTraceState(tCtx.GetSpan().TraceState().AsRaw()); err == nil {
					s, err := keys[0].String(ctx, tCtx)
					if err != nil {
						return err
					}
					if s == nil {
						return fmt.Errorf("trace_state indexing type must be a string")
					}
					if updated, err := ts.Insert(*s, str); err == nil {
						tCtx.GetSpan().TraceState().FromRaw(updated.String())
					}
				}
			}
			return nil
		},
	}, nil
}

func accessParentSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().ParentSpanID(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if newParentSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetSpan().SetParentSpanID(newParentSpanID)
			}
			return nil
		},
	}
}

func accessStringParentSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			id := tCtx.GetSpan().ParentSpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				id, err := ParseSpanID(str)
				if err != nil {
					return err
				}
				tCtx.GetSpan().SetParentSpanID(id)
			}
			return nil
		},
	}
}

func accessSpanName[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Name(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().SetName(str)
			}
			return nil
		},
	}
}

func accessKind[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpan().Kind()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetKind(ptrace.SpanKind(i))
			}
			return nil
		},
	}
}

func accessStringKind[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Kind().String(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if s, ok := val.(string); ok {
				var kind ptrace.SpanKind
				switch s {
				case "Unspecified":
					kind = ptrace.SpanKindUnspecified
				case "Internal":
					kind = ptrace.SpanKindInternal
				case "Server":
					kind = ptrace.SpanKindServer
				case "Client":
					kind = ptrace.SpanKindClient
				case "Producer":
					kind = ptrace.SpanKindProducer
				case "Consumer":
					kind = ptrace.SpanKindConsumer
				default:
					return fmt.Errorf("unknown span kind string, %v", s)
				}
				tCtx.GetSpan().SetKind(kind)
			}
			return nil
		},
	}
}

func accessDeprecatedStringKind[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return traceutil.SpanKindStr(tCtx.GetSpan().Kind()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if s, ok := val.(string); ok {
				var kind ptrace.SpanKind
				switch s {
				case "SPAN_KIND_UNSPECIFIED":
					kind = ptrace.SpanKindUnspecified
				case "SPAN_KIND_INTERNAL":
					kind = ptrace.SpanKindInternal
				case "SPAN_KIND_SERVER":
					kind = ptrace.SpanKindServer
				case "SPAN_KIND_CLIENT":
					kind = ptrace.SpanKindClient
				case "SPAN_KIND_PRODUCER":
					kind = ptrace.SpanKindProducer
				case "SPAN_KIND_CONSUMER":
					kind = ptrace.SpanKindConsumer
				default:
					return fmt.Errorf("unknown span kind deprecated string, %v", s)
				}
				tCtx.GetSpan().SetKind(kind)
			}
			return nil
		},
	}
}

func accessStartTimeUnixNano[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().StartTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessEndTimeUnixNano[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().EndTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetEndTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessStartTime[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().StartTimestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetSpan().SetStartTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessEndTime[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().EndTimestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetSpan().SetEndTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessAttributes[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetSpan().Attributes())
			}
			return nil
		},
	}
}

func accessAttributesKey[K SpanContext](keys []ottl.Key[K]) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (any, error) {
			return GetMapValue[K](ctx, tCtx, tCtx.GetSpan().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx K, val any) error {
			return SetMapValue[K](ctx, tCtx, tCtx.GetSpan().Attributes(), keys, val)
		},
	}
}

func accessSpanDroppedAttributesCount[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpan().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessEvents[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Events(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if slc, ok := val.(ptrace.SpanEventSlice); ok {
				tCtx.GetSpan().Events().RemoveIf(func(_ ptrace.SpanEvent) bool {
					return true
				})
				slc.CopyTo(tCtx.GetSpan().Events())
			}
			return nil
		},
	}
}

func accessDroppedEventsCount[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpan().DroppedEventsCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedEventsCount(uint32(i))
			}
			return nil
		},
	}
}

func accessLinks[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Links(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if slc, ok := val.(ptrace.SpanLinkSlice); ok {
				tCtx.GetSpan().Links().RemoveIf(func(_ ptrace.SpanLink) bool {
					return true
				})
				slc.CopyTo(tCtx.GetSpan().Links())
			}
			return nil
		},
	}
}

func accessDroppedLinksCount[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpan().DroppedLinksCount()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedLinksCount(uint32(i))
			}
			return nil
		},
	}
}

func accessStatus[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Status(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if status, ok := val.(ptrace.Status); ok {
				status.CopyTo(tCtx.GetSpan().Status())
			}
			return nil
		},
	}
}

func accessStatusCode[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return int64(tCtx.GetSpan().Status().Code()), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().Status().SetCode(ptrace.StatusCode(i))
			}
			return nil
		},
	}
}

func accessStatusMessage[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(_ context.Context, tCtx K) (any, error) {
			return tCtx.GetSpan().Status().Message(), nil
		},
		Setter: func(_ context.Context, tCtx K, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().Status().SetMessage(str)
			}
			return nil
		},
	}
}
