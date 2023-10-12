// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottllog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

var _ internal.ResourceContext = TransformContext{}
var _ internal.InstrumentationScopeContext = TransformContext{}

type TransformContext struct {
	logRecord            plog.LogRecord
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(logRecord plog.LogRecord, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource) TransformContext {
	return TransformContext{
		logRecord:            logRecord,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
	}
}

func (tCtx TransformContext) GetLogRecord() plog.LogRecord {
	return tCtx.logRecord
}

func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

func (tCtx TransformContext) getCache() pcommon.Map {
	return tCtx.cache
}

func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
	p, err := ottl.NewParser[TransformContext](
		functions,
		parsePath,
		telemetrySettings,
		ottl.WithEnumParser[TransformContext](parseEnum),
	)
	if err != nil {
		return ottl.Parser[TransformContext]{}, err
	}
	for _, opt := range options {
		opt(&p)
	}
	return p, nil
}

type StatementsOption func(*ottl.Statements[TransformContext])

func WithErrorMode(errorMode ottl.ErrorMode) StatementsOption {
	return func(s *ottl.Statements[TransformContext]) {
		ottl.WithErrorMode[TransformContext](errorMode)(s)
	}
}

func NewStatements(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementsOption) ottl.Statements[TransformContext] {
	s := ottl.NewStatements(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

var symbolTable = map[ottl.EnumSymbol]ottl.Enum{
	"SEVERITY_NUMBER_UNSPECIFIED": ottl.Enum(plog.SeverityNumberUnspecified),
	"SEVERITY_NUMBER_TRACE":       ottl.Enum(plog.SeverityNumberTrace),
	"SEVERITY_NUMBER_TRACE2":      ottl.Enum(plog.SeverityNumberTrace2),
	"SEVERITY_NUMBER_TRACE3":      ottl.Enum(plog.SeverityNumberTrace3),
	"SEVERITY_NUMBER_TRACE4":      ottl.Enum(plog.SeverityNumberTrace4),
	"SEVERITY_NUMBER_DEBUG":       ottl.Enum(plog.SeverityNumberDebug),
	"SEVERITY_NUMBER_DEBUG2":      ottl.Enum(plog.SeverityNumberDebug2),
	"SEVERITY_NUMBER_DEBUG3":      ottl.Enum(plog.SeverityNumberDebug3),
	"SEVERITY_NUMBER_DEBUG4":      ottl.Enum(plog.SeverityNumberDebug4),
	"SEVERITY_NUMBER_INFO":        ottl.Enum(plog.SeverityNumberInfo),
	"SEVERITY_NUMBER_INFO2":       ottl.Enum(plog.SeverityNumberInfo2),
	"SEVERITY_NUMBER_INFO3":       ottl.Enum(plog.SeverityNumberInfo3),
	"SEVERITY_NUMBER_INFO4":       ottl.Enum(plog.SeverityNumberInfo4),
	"SEVERITY_NUMBER_WARN":        ottl.Enum(plog.SeverityNumberWarn),
	"SEVERITY_NUMBER_WARN2":       ottl.Enum(plog.SeverityNumberWarn2),
	"SEVERITY_NUMBER_WARN3":       ottl.Enum(plog.SeverityNumberWarn3),
	"SEVERITY_NUMBER_WARN4":       ottl.Enum(plog.SeverityNumberWarn4),
	"SEVERITY_NUMBER_ERROR":       ottl.Enum(plog.SeverityNumberError),
	"SEVERITY_NUMBER_ERROR2":      ottl.Enum(plog.SeverityNumberError2),
	"SEVERITY_NUMBER_ERROR3":      ottl.Enum(plog.SeverityNumberError3),
	"SEVERITY_NUMBER_ERROR4":      ottl.Enum(plog.SeverityNumberError4),
	"SEVERITY_NUMBER_FATAL":       ottl.Enum(plog.SeverityNumberFatal),
	"SEVERITY_NUMBER_FATAL2":      ottl.Enum(plog.SeverityNumberFatal2),
	"SEVERITY_NUMBER_FATAL3":      ottl.Enum(plog.SeverityNumberFatal3),
	"SEVERITY_NUMBER_FATAL4":      ottl.Enum(plog.SeverityNumberFatal4),
}

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		if enum, ok := symbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, fmt.Errorf("enum symbol not provided")
}

func parsePath(val *ottl.Path) (ottl.GetSetter[TransformContext], error) {
	if val != nil && len(val.Fields) > 0 {
		return newPathGetSetter(val.Fields)
	}
	return nil, fmt.Errorf("bad path %v", val)
}

func newPathGetSetter(path []ottl.Field) (ottl.GetSetter[TransformContext], error) {
	switch path[0].Name {
	case "cache":
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessCache(), nil
		}
		return accessCacheKey(mapKey), nil
	case "resource":
		return internal.ResourcePathGetSetter[TransformContext](path[1:])
	case "instrumentation_scope":
		return internal.ScopePathGetSetter[TransformContext](path[1:])
	case "time_unix_nano":
		return accessTimeUnixNano(), nil
	case "observed_time_unix_nano":
		return accessObservedTimeUnixNano(), nil
	case "severity_number":
		return accessSeverityNumber(), nil
	case "severity_text":
		return accessSeverityText(), nil
	case "body":
		if len(path) == 1 {
			keys := path[0].Keys
			if keys == nil {
				return accessBody(), nil
			}
			return accessBodyKey(keys), nil
		}
		if path[1].Name == "string" {
			return accessStringBody(), nil
		}
	case "attributes":
		mapKey := path[0].Keys
		if mapKey == nil {
			return accessAttributes(), nil
		}
		return accessAttributesKey(mapKey), nil
	case "dropped_attributes_count":
		return accessDroppedAttributesCount(), nil
	case "flags":
		return accessFlags(), nil
	case "trace_id":
		if len(path) == 1 {
			return accessTraceID(), nil
		}
		if path[1].Name == "string" {
			return accessStringTraceID(), nil
		}
	case "span_id":
		if len(path) == 1 {
			return accessSpanID(), nil
		}
		if path[1].Name == "string" {
			return accessStringSpanID(), nil
		}
	}

	return nil, fmt.Errorf("invalid path expression %v", path)
}

func accessCache() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.getCache(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if m, ok := val.(pcommon.Map); ok {
				m.CopyTo(tCtx.getCache())
			}
			return nil
		},
	}
}

func accessCacheKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return internal.GetMapValue(tCtx.getCache(), keys)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetMapValue(tCtx.getCache(), keys, val)
		},
	}
}

func accessTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessObservedTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().ObservedTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessSeverityNumber() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return int64(tCtx.GetLogRecord().SeverityNumber()), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetSeverityNumber(plog.SeverityNumber(i))
			}
			return nil
		},
	}
}

func accessSeverityText() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().SeverityText(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if s, ok := val.(string); ok {
				tCtx.GetLogRecord().SetSeverityText(s)
			}
			return nil
		},
	}
}

func accessBody() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return ottlcommon.GetValue(tCtx.GetLogRecord().Body()), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetValue(tCtx.GetLogRecord().Body(), val)
		},
	}
}

func accessBodyKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return internal.GetMapValue(tCtx.GetLogRecord().Body().Map(), keys)
			case pcommon.ValueTypeSlice:
				return internal.GetSliceValue(tCtx.GetLogRecord().Body().Slice(), keys)
			default:
				return nil, fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return internal.SetMapValue(tCtx.GetLogRecord().Body().Map(), keys, val)
			case pcommon.ValueTypeSlice:
				return internal.SetSliceValue(tCtx.GetLogRecord().Body().Slice(), keys, val)
			default:
				return fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
	}
}

func accessStringBody() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().Body().AsString(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if str, ok := val.(string); ok {
				tCtx.GetLogRecord().Body().SetStr(str)
			}
			return nil
		},
	}
}

func accessAttributes() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().Attributes(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetLogRecord().Attributes())
			}
			return nil
		},
	}
}

func accessAttributesKey(keys []ottl.Key) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return internal.GetMapValue(tCtx.GetLogRecord().Attributes(), keys)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			return internal.SetMapValue(tCtx.GetLogRecord().Attributes(), keys, val)
		},
	}
}

func accessDroppedAttributesCount() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return int64(tCtx.GetLogRecord().DroppedAttributesCount()), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessFlags() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return int64(tCtx.GetLogRecord().Flags()), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetFlags(plog.LogRecordFlags(i))
			}
			return nil
		},
	}
}

func accessTraceID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().TraceID(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newTraceID, ok := val.(pcommon.TraceID); ok {
				tCtx.GetLogRecord().SetTraceID(newTraceID)
			}
			return nil
		},
	}
}

func accessStringTraceID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			id := tCtx.GetLogRecord().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if str, ok := val.(string); ok {
				id, err := internal.ParseTraceID(str)
				if err != nil {
					return err
				}
				tCtx.GetLogRecord().SetTraceID(id)
			}
			return nil
		},
	}
}

func accessSpanID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			return tCtx.GetLogRecord().SpanID(), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if newSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetLogRecord().SetSpanID(newSpanID)
			}
			return nil
		},
	}
}

func accessStringSpanID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (interface{}, error) {
			id := tCtx.GetLogRecord().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val interface{}) error {
			if str, ok := val.(string); ok {
				id, err := internal.ParseSpanID(str)
				if err != nil {
					return err
				}
				tCtx.GetLogRecord().SetSpanID(id)
			}
			return nil
		},
	}
}
