// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottllog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxlog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

var tcPool = sync.Pool{
	New: func() any {
		return &TransformContext{cache: pcommon.NewMap()}
	},
}

// ContextName is the name of the context for logs.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxlog.Name

var (
	_ ctxresource.Context     = (*TransformContext)(nil)
	_ ctxscope.Context        = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler = (*TransformContext)(nil)
)

// TransformContext represents a log and its associated hierarchy.
type TransformContext struct {
	resourceLogs plog.ResourceLogs
	scopeLogs    plog.ScopeLogs
	logRecord    plog.LogRecord
	cache        pcommon.Map
}

type logRecord plog.LogRecord

// MarshalLogObject serializes the log into a zapcore.ObjectEncoder for logging.
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
	encoder.AddString("event_name", lr.EventName())
	return err
}

// MarshalLogObject serializes the TransformContext into a zapcore.ObjectEncoder for logging.
func (tCtx *TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.GetResource()))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.GetInstrumentationScope())))
	err = errors.Join(err, encoder.AddObject("log_record", logRecord(tCtx.logRecord)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// NewTransformContextPtr returns a new TransformContext with the provided parameters from a pool of contexts.
// Caller must call TransformContext.Close on the returned TransformContext.
func NewTransformContextPtr(resourceLogs plog.ResourceLogs, scopeLogs plog.ScopeLogs, logRecord plog.LogRecord, options ...TransformContextOption) *TransformContext {
	tCtx := tcPool.Get().(*TransformContext)
	tCtx.resourceLogs = resourceLogs
	tCtx.scopeLogs = scopeLogs
	tCtx.logRecord = logRecord
	for _, opt := range options {
		opt(tCtx)
	}
	return tCtx
}

// Close the current TransformContext.
// After this function returns this instance cannot be used.
func (tCtx *TransformContext) Close() {
	tCtx.resourceLogs = plog.ResourceLogs{}
	tCtx.scopeLogs = plog.ScopeLogs{}
	tCtx.logRecord = plog.LogRecord{}
	tCtx.cache.Clear()
	tcPool.Put(tCtx)
}

// GetLogRecord returns the log record from the TransformContext.
func (tCtx *TransformContext) GetLogRecord() plog.LogRecord {
	return tCtx.logRecord
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx *TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.scopeLogs.Scope()
}

// GetResource returns the resource from the TransformContext.
func (tCtx *TransformContext) GetResource() pcommon.Resource {
	return tCtx.resourceLogs.Resource()
}

// GetScopeSchemaURLItem returns the scope schema URL item from the TransformContext.
func (tCtx *TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.scopeLogs
}

// GetResourceSchemaURLItem returns the resource schema URL item from the TransformContext.
func (tCtx *TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.resourceLogs
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[*TransformContext] {
	return func(p *ottl.Parser[*TransformContext]) {
		ottl.WithPathContextNames[*TransformContext]([]string{
			ctxlog.Name,
			ctxscope.LegacyName,
			ctxscope.Name,
			ctxresource.Name,
		})(p)
	}
}

// StatementSequenceOption represents an option for configuring a statement sequence.
type StatementSequenceOption func(*ottl.StatementSequence[*TransformContext])

// WithStatementSequenceErrorMode sets the error mode for a statement sequence.
func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[*TransformContext]) {
		ottl.WithStatementSequenceErrorMode[*TransformContext](errorMode)(s)
	}
}

// NewStatementSequence creates a new statement sequence with the provided statements and options.
func NewStatementSequence(statements []*ottl.Statement[*TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[*TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

// ConditionSequenceOption represents an option for configuring a condition sequence.
type ConditionSequenceOption func(*ottl.ConditionSequence[*TransformContext])

// WithConditionSequenceErrorMode sets the error mode for a condition sequence.
func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[*TransformContext]) {
		ottl.WithConditionSequenceErrorMode[*TransformContext](errorMode)(c)
	}
}

// NewConditionSequence creates a new condition sequence with the provided conditions and options.
func NewConditionSequence(conditions []*ottl.Condition[*TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[*TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
}

// NewParser creates a new log parser with the provided functions and options.
func NewParser(
	functions map[string]ottl.Factory[*TransformContext],
	telemetrySettings component.TelemetrySettings,
	options ...ottl.Option[*TransformContext],
) (ottl.Parser[*TransformContext], error) {
	return ctxcommon.NewParser(
		functions,
		telemetrySettings,
		pathExpressionParser(getCache),
		parseEnum,
		options...,
	)
}

func parseEnum(val *ottl.EnumSymbol) (*ottl.Enum, error) {
	if val != nil {
		if enum, ok := ctxlog.SymbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, errors.New("enum symbol not provided")
}

func getCache(tCtx *TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[*TransformContext]) ottl.PathExpressionParser[*TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxlog.Name,
		ctxlog.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[*TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[*TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[*TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[*TransformContext],
			ctxlog.Name:         ctxlog.PathGetSetter[*TransformContext],
		})
}
