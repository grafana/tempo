// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ottlcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ottlcommon"

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/trace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
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

func SpanPathGetSetter[K SpanContext](path []ottl.Field) (ottl.GetSetter[K], error) {
	if len(path) == 0 {
		return accessSpan[K](), nil
	}

	switch path[0].Name {
	case "trace_id":
		if len(path) == 1 {
			return accessTraceID[K](), nil
		}
		if path[1].Name == "string" {
			return accessStringTraceID[K](), nil
		}
	case "span_id":
		if len(path) == 1 {
			return accessSpanID[K](), nil
		}
		if path[1].Name == "string" {
			return accessStringSpanID[K](), nil
		}
	case "trace_state":
		mapKey := path[0].MapKey
		if mapKey == nil {
			return accessTraceState[K](), nil
		}
		return accessTraceStateKey[K](mapKey), nil
	case "parent_span_id":
		if len(path) == 1 {
			return accessParentSpanID[K](), nil
		}
		if path[1].Name == "string" {
			return accessStringParentSpanID[K](), nil
		}
	case "name":
		return accessSpanName[K](), nil
	case "kind":
		return accessKind[K](), nil
	case "start_time_unix_nano":
		return accessStartTimeUnixNano[K](), nil
	case "end_time_unix_nano":
		return accessEndTimeUnixNano[K](), nil
	case "attributes":
		mapKey := path[0].MapKey
		if mapKey == nil {
			return accessAttributes[K](), nil
		}
		return accessAttributesKey[K](mapKey), nil
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
		if len(path) == 1 {
			return accessStatus[K](), nil
		}
		switch path[1].Name {
		case "code":
			return accessStatusCode[K](), nil
		case "message":
			return accessStatusMessage[K](), nil
		}
	}

	return nil, fmt.Errorf("invalid span path expression %v", path)
}

func accessSpan[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newSpan, ok := val.(ptrace.Span); ok {
				newSpan.CopyTo(tCtx.GetSpan())
			}
			return nil
		},
	}
}

func accessTraceID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().TraceID(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newTraceID, ok := val.(pcommon.TraceID); ok {
				tCtx.GetSpan().SetTraceID(newTraceID)
			}
			return nil
		},
	}
}

func accessStringTraceID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			id := tCtx.GetSpan().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
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
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().SpanID(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetSpan().SetSpanID(newSpanID)
			}
			return nil
		},
	}
}

func accessStringSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			id := tCtx.GetSpan().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
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
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().TraceState().AsRaw(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().TraceState().FromRaw(str)
			}
			return nil
		},
	}
}

func accessTraceStateKey[K SpanContext](mapKey *string) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			if ts, err := trace.ParseTraceState(tCtx.GetSpan().TraceState().AsRaw()); err == nil {
				return ts.Get(*mapKey), nil
			}
			return nil, nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				if ts, err := trace.ParseTraceState(tCtx.GetSpan().TraceState().AsRaw()); err == nil {
					if updated, err := ts.Insert(*mapKey, str); err == nil {
						tCtx.GetSpan().TraceState().FromRaw(updated.String())
					}
				}
			}
			return nil
		},
	}
}

func accessParentSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().ParentSpanID(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if newParentSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetSpan().SetParentSpanID(newParentSpanID)
			}
			return nil
		},
	}
}

func accessStringParentSpanID[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			id := tCtx.GetSpan().ParentSpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
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
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Name(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().SetName(str)
			}
			return nil
		},
	}
}

func accessKind[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetSpan().Kind()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetKind(ptrace.SpanKind(i))
			}
			return nil
		},
	}
}

func accessStartTimeUnixNano[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().StartTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessEndTimeUnixNano[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().EndTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetEndTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessAttributes[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetSpan().Attributes())
			}
			return nil
		},
	}
}

func accessAttributesKey[K SpanContext](mapKey *string) ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return GetMapValue(tCtx.GetSpan().Attributes(), *mapKey), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			SetMapValue(tCtx.GetSpan().Attributes(), *mapKey, val)
			return nil
		},
	}
}

func accessSpanDroppedAttributesCount[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetSpan().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessEvents[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Events(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if slc, ok := val.(ptrace.SpanEventSlice); ok {
				tCtx.GetSpan().Events().RemoveIf(func(event ptrace.SpanEvent) bool {
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
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetSpan().DroppedEventsCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedEventsCount(uint32(i))
			}
			return nil
		},
	}
}

func accessLinks[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Links(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if slc, ok := val.(ptrace.SpanLinkSlice); ok {
				tCtx.GetSpan().Links().RemoveIf(func(event ptrace.SpanLink) bool {
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
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetSpan().DroppedLinksCount()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().SetDroppedLinksCount(uint32(i))
			}
			return nil
		},
	}
}

func accessStatus[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Status(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if status, ok := val.(ptrace.Status); ok {
				status.CopyTo(tCtx.GetSpan().Status())
			}
			return nil
		},
	}
}

func accessStatusCode[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return int64(tCtx.GetSpan().Status().Code()), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetSpan().Status().SetCode(ptrace.StatusCode(i))
			}
			return nil
		},
	}
}

func accessStatusMessage[K SpanContext]() ottl.StandardGetSetter[K] {
	return ottl.StandardGetSetter[K]{
		Getter: func(ctx context.Context, tCtx K) (interface{}, error) {
			return tCtx.GetSpan().Status().Message(), nil
		},
		Setter: func(ctx context.Context, tCtx K, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetSpan().Status().SetMessage(str)
			}
			return nil
		},
	}
}
