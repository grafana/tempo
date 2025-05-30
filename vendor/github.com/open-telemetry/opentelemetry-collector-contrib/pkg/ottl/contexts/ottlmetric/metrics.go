// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlmetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

// ContextName is the name of the context for metrics.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxmetric.Name

var (
	_ ctxresource.Context = TransformContext{}
	_ ctxscope.Context    = TransformContext{}
	_ ctxmetric.Context   = TransformContext{}
)

// TransformContext represents a metric and its associated hierarchy.
type TransformContext struct {
	metric               pmetric.Metric
	metrics              pmetric.MetricSlice
	instrumentationScope pcommon.InstrumentationScope
	resource             pcommon.Resource
	cache                pcommon.Map
	scopeMetrics         pmetric.ScopeMetrics
	resourceMetrics      pmetric.ResourceMetrics
}

// MarshalLogObject serializes the metric into a zapcore.ObjectEncoder for logging.
func (tCtx TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.resource))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.instrumentationScope)))
	err = errors.Join(err, encoder.AddObject("metric", logging.Metric(tCtx.metric)))
	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))

	return err
}

// TransformContextOption represents an option for configuring a TransformContext.
type TransformContextOption func(*TransformContext)

// NewTransformContext creates a new TransformContext with the provided parameters.
func NewTransformContext(metric pmetric.Metric, metrics pmetric.MetricSlice, instrumentationScope pcommon.InstrumentationScope, resource pcommon.Resource, scopeMetrics pmetric.ScopeMetrics, resourceMetrics pmetric.ResourceMetrics, options ...TransformContextOption) TransformContext {
	tc := TransformContext{
		metric:               metric,
		metrics:              metrics,
		instrumentationScope: instrumentationScope,
		resource:             resource,
		cache:                pcommon.NewMap(),
		scopeMetrics:         scopeMetrics,
		resourceMetrics:      resourceMetrics,
	}
	for _, opt := range options {
		opt(&tc)
	}
	return tc
}

// GetMetric returns the metric from the TransformContext.
func (tCtx TransformContext) GetMetric() pmetric.Metric {
	return tCtx.metric
}

// GetMetrics returns the metric slice from the TransformContext.
func (tCtx TransformContext) GetMetrics() pmetric.MetricSlice {
	return tCtx.metrics
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.instrumentationScope
}

// GetResource returns the resource from the TransformContext.
func (tCtx TransformContext) GetResource() pcommon.Resource {
	return tCtx.resource
}

// GetScopeSchemaURLItem returns the scope schema URL item from the TransformContext.
func (tCtx TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.scopeMetrics
}

// GetResourceSchemaURLItem returns the resource schema URL item from the TransformContext.
func (tCtx TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.resourceMetrics
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[TransformContext] {
	return func(p *ottl.Parser[TransformContext]) {
		ottl.WithPathContextNames[TransformContext]([]string{
			ctxmetric.Name,
			ctxscope.LegacyName,
			ctxresource.Name,
		})(p)
	}
}

// StatementSequenceOption represents an option for configuring a statement sequence.
type StatementSequenceOption func(*ottl.StatementSequence[TransformContext])

// WithStatementSequenceErrorMode sets the error mode for a statement sequence.
func WithStatementSequenceErrorMode(errorMode ottl.ErrorMode) StatementSequenceOption {
	return func(s *ottl.StatementSequence[TransformContext]) {
		ottl.WithStatementSequenceErrorMode[TransformContext](errorMode)(s)
	}
}

// NewStatementSequence creates a new statement sequence with the provided statements and options.
func NewStatementSequence(statements []*ottl.Statement[TransformContext], telemetrySettings component.TelemetrySettings, options ...StatementSequenceOption) ottl.StatementSequence[TransformContext] {
	s := ottl.NewStatementSequence(statements, telemetrySettings)
	for _, op := range options {
		op(&s)
	}
	return s
}

// ConditionSequenceOption represents an option for configuring a condition sequence.
type ConditionSequenceOption func(*ottl.ConditionSequence[TransformContext])

// WithConditionSequenceErrorMode sets the error mode for a condition sequence.
func WithConditionSequenceErrorMode(errorMode ottl.ErrorMode) ConditionSequenceOption {
	return func(c *ottl.ConditionSequence[TransformContext]) {
		ottl.WithConditionSequenceErrorMode[TransformContext](errorMode)(c)
	}
}

// NewConditionSequence creates a new condition sequence with the provided conditions and options.
func NewConditionSequence(conditions []*ottl.Condition[TransformContext], telemetrySettings component.TelemetrySettings, options ...ConditionSequenceOption) ottl.ConditionSequence[TransformContext] {
	c := ottl.NewConditionSequence(conditions, telemetrySettings)
	for _, op := range options {
		op(&c)
	}
	return c
}

// NewParser creates a new metric parser with the provided functions and options.
func NewParser(
	functions map[string]ottl.Factory[TransformContext],
	telemetrySettings component.TelemetrySettings,
	options ...ottl.Option[TransformContext],
) (ottl.Parser[TransformContext], error) {
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
		if enum, ok := ctxmetric.SymbolTable[*val]; ok {
			return &enum, nil
		}
		return nil, fmt.Errorf("enum symbol, %s, not found", *val)
	}
	return nil, errors.New("enum symbol not provided")
}

func getCache(tCtx TransformContext) pcommon.Map {
	return tCtx.cache
}

func pathExpressionParser(cacheGetter ctxcache.Getter[TransformContext]) ottl.PathExpressionParser[TransformContext] {
	return ctxcommon.PathExpressionParser(
		ctxmetric.Name,
		ctxmetric.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[TransformContext],
			ctxmetric.Name:      ctxmetric.PathGetSetter[TransformContext],
		})
}
