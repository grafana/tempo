// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottllog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
	common "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/internal/ottlcommon"
)

const (
	contextName = "Log"
)

var (
	_ internal.ResourceContext             = (*TransformContext)(nil)
	_ internal.InstrumentationScopeContext = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler              = (*TransformContext)(nil)
)

type TransformContext struct {
	logRecord            plog.LogRecord
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
	scopeLogs            plog.ScopeLogs
	resourceLogs         plog.ResourceLogs
}

type logRecord plog.LogRecord

func (l logRecord) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	lr := plog.LogRecord(l)
	spanID := lr.SpanID()
	traceID := lr.TraceID()
	err := encoder.AddObject("attributes", logging.Map(lr.Attributes()))
	encoder.AddString("body", lr.Body().AsString())
	encoder.AddUint32("dropped_attribute_count", lr.DroppedAttributesCount())
	encoder.AddUint32("flags", uint32(lr.Flags()))
	encoder.AddUint64("observed_time_unix_nano", uint64(lr.ObservedTimestamp()))
	encoder.AddInt32("severity_number", int32(lr.SeverityNumber()))
	encoder.AddString("severity_text", lr.SeverityText())
	encoder.AddString("span_id", hex.EncodeToString(spanID[:]))
	encoder.AddUint64("time_unix_nano", uint64(lr.Timestamp()))
	encoder.AddString("trace_id", hex.EncodeToString(traceID[:]))
	return err
}

func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.instrumentationScope)))
	err = errors.Join(err, encoder.AddObject("log_record", logRecord(tCtx.logRecord)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

type Option func(*ottl.Parser[TransformContext])

func NewTransformContext(logRecord plog.LogRecord, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource, scopeLogs plog.ScopeLogs, resourceLogs plog.ResourceLogs) TransformContext {
	return TransformContext{
		logRecord:            logRecord,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
		scopeLogs:            scopeLogs,
		resourceLogs:         resourceLogs,
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

func (tCtx TransformContext) GetScopeSchemaURLItem() internal.SchemaURLItem {
	return tCtx.scopeLogs
}

func (tCtx TransformContext) GetResourceSchemaURLItem() internal.SchemaURLItem {
	return tCtx.resourceLogs
}

func NewParser(functions map[string]ottl.Factory[TransformContext], telemetrySettings component.TelemetrySettings, options ...Option) (ottl.Parser[TransformContext], error) {
	pep := pathExpressionParser{telemetrySettings}
	p, err := ottl.NewParser[TransformContext](
		functions,
		pep.parsePath,
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

type StatementSequenceOption func(*ottl.StatementSequence[TransformContext])

func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[TransformContext]) {
		ottl.WithStatementSequenceErrorMode[TransformContext](errorMode)(s)
	}
}

func NewStatementSequence(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

type ConditionSequenceOption func(*ottl.ConditionSequence[TransformContext])

func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[TransformContext]) {
		ottl.WithConditionSequenceErrorMode[TransformContext](errorMode)(c)
	}
}

func NewConditionSequence(conditions []*ottl.Condition[TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
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

type pathExpressionParser struct {
	telemetrySettings component.TelemetrySettings
}

func (pep *pathExpressionParser) parsePath(path ottl.Path[TransformContext]) (ottl.GetSetter[TransformContext], error) {
	if path == nil {
		return nil, fmt.Errorf("path cannot be nil")
	}
	switch path.Name() {
	case "cache":
		if path.Keys() == nil {
			return accessCache(), nil
		}
		return accessCacheKey(path.Keys()), nil
	case "resource":
		return internal.ResourcePathGetSetter[TransformContext](path.Next())
	case "instrumentation_scope":
		return internal.ScopePathGetSetter[TransformContext](path.Next())
	case "time_unix_nano":
		return accessTimeUnixNano(), nil
	case "observed_time_unix_nano":
		return accessObservedTimeUnixNano(), nil
	case "time":
		return accessTime(), nil
	case "observed_time":
		return accessObservedTime(), nil
	case "severity_number":
		return accessSeverityNumber(), nil
	case "severity_text":
		return accessSeverityText(), nil
	case "body":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringBody(), nil
			}
			return nil, internal.FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), contextName, internal.LogRef)
		}
		if path.Keys() == nil {
			return accessBody(), nil
		}
		return accessBodyKey(path.Keys()), nil
	case "attributes":
		if path.Keys() == nil {
			return accessAttributes(), nil
		}
		return accessAttributesKey(path.Keys()), nil
	case "dropped_attributes_count":
		return accessDroppedAttributesCount(), nil
	case "flags":
		return accessFlags(), nil
	case "trace_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringTraceID(), nil
			}
			return nil, internal.FormatDefaultErrorMessage(nextPath.Name(), nextPath.String(), contextName, internal.LogRef)
		}
		return accessTraceID(), nil
	case "span_id":
		nextPath := path.Next()
		if nextPath != nil {
			if nextPath.Name() == "string" {
				return accessStringSpanID(), nil
			}
			return nil, internal.FormatDefaultErrorMessage(nextPath.Name(), path.String(), contextName, internal.LogRef)
		}
		return accessSpanID(), nil
	default:
		return nil, internal.FormatDefaultErrorMessage(path.Name(), path.String(), contextName, internal.LogRef)
	}
}

func accessCache() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.getCache(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if m, ok := val.(pcommon.Map); ok {
				m.CopyTo(tCtx.getCache())
			}
			return nil
		},
	}
}

func accessCacheKey(key []ottl.Key[TransformContext]) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (any, error) {
			return internal.GetMapValue[TransformContext](ctx, tCtx, tCtx.getCache(), key)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val any) error {
			return internal.SetMapValue[TransformContext](ctx, tCtx, tCtx.getCache(), key, val)
		},
	}
}

func accessTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().Timestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessObservedTimeUnixNano() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().ObservedTimestamp().AsTime().UnixNano(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, i)))
			}
			return nil
		},
	}
}

func accessTime() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().Timestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetLogRecord().SetTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessObservedTime() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().ObservedTimestamp().AsTime(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(time.Time); ok {
				tCtx.GetLogRecord().SetObservedTimestamp(pcommon.NewTimestampFromTime(i))
			}
			return nil
		},
	}
}

func accessSeverityNumber() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return int64(tCtx.GetLogRecord().SeverityNumber()), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetSeverityNumber(plog.SeverityNumber(i))
			}
			return nil
		},
	}
}

func accessSeverityText() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().SeverityText(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if s, ok := val.(string); ok {
				tCtx.GetLogRecord().SetSeverityText(s)
			}
			return nil
		},
	}
}

func accessBody() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return common.GetValue(tCtx.GetLogRecord().Body()), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			return internal.SetValue(tCtx.GetLogRecord().Body(), val)
		},
	}
}

func accessBodyKey(key []ottl.Key[TransformContext]) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (any, error) {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return internal.GetMapValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Body().Map(), key)
			case pcommon.ValueTypeSlice:
				return internal.GetSliceValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Body().Slice(), key)
			default:
				return nil, fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val any) error {
			body := tCtx.GetLogRecord().Body()
			switch body.Type() {
			case pcommon.ValueTypeMap:
				return internal.SetMapValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Body().Map(), key, val)
			case pcommon.ValueTypeSlice:
				return internal.SetSliceValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Body().Slice(), key, val)
			default:
				return fmt.Errorf("log bodies of type %s cannot be indexed", body.Type().String())
			}
		},
	}
}

func accessStringBody() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().Body().AsString(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if str, ok := val.(string); ok {
				tCtx.GetLogRecord().Body().SetStr(str)
			}
			return nil
		},
	}
}

func accessAttributes() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().Attributes(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if attrs, ok := val.(pcommon.Map); ok {
				attrs.CopyTo(tCtx.GetLogRecord().Attributes())
			}
			return nil
		},
	}
}

func accessAttributesKey(key []ottl.Key[TransformContext]) ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(ctx context.Context, tCtx TransformContext) (any, error) {
			return internal.GetMapValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Attributes(), key)
		},
		Setter: func(ctx context.Context, tCtx TransformContext, val any) error {
			return internal.SetMapValue[TransformContext](ctx, tCtx, tCtx.GetLogRecord().Attributes(), key, val)
		},
	}
}

func accessDroppedAttributesCount() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return int64(tCtx.GetLogRecord().DroppedAttributesCount()), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetDroppedAttributesCount(uint32(i))
			}
			return nil
		},
	}
}

func accessFlags() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return int64(tCtx.GetLogRecord().Flags()), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if i, ok := val.(int64); ok {
				tCtx.GetLogRecord().SetFlags(plog.LogRecordFlags(i))
			}
			return nil
		},
	}
}

func accessTraceID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().TraceID(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if newTraceID, ok := val.(pcommon.TraceID); ok {
				tCtx.GetLogRecord().SetTraceID(newTraceID)
			}
			return nil
		},
	}
}

func accessStringTraceID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			id := tCtx.GetLogRecord().TraceID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
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
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			return tCtx.GetLogRecord().SpanID(), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
			if newSpanID, ok := val.(pcommon.SpanID); ok {
				tCtx.GetLogRecord().SetSpanID(newSpanID)
			}
			return nil
		},
	}
}

func accessStringSpanID() ottl.StandardGetSetter[TransformContext] {
	return ottl.StandardGetSetter[TransformContext]{
		Getter: func(_ context.Context, tCtx TransformContext) (any, error) {
			id := tCtx.GetLogRecord().SpanID()
			return hex.EncodeToString(id[:]), nil
		},
		Setter: func(_ context.Context, tCtx TransformContext, val any) error {
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
