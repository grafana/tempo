// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottldatapoint // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxdatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logging"
)

var tcPool = sync.Pool{
	New: func() any {
		return &TransformContext{cache: pcommon.NewMap()}
	},
}

// ContextName is the name of the context for datapoints.
// Experimental: *NOTE* this constant is subject to change or removal in the future.
const ContextName = ctxdatapoint.Name

var (
	_ ctxresource.Context     = (*TransformContext)(nil)
	_ ctxscope.Context        = (*TransformContext)(nil)
	_ ctxmetric.Context       = (*TransformContext)(nil)
	_ zapcore.ObjectMarshaler = (*TransformContext)(nil)
)

// TransformContext represents a Datapoint and all its hierarchy.
type TransformContext struct {
	resourceMetrics pmetric.ResourceMetrics
	scopeMetrics    pmetric.ScopeMetrics
	metric          pmetric.Metric
	dataPoint       any
	cache           pcommon.Map
}

// MarshalLogObject serializes the TransformContext into a zapcore.ObjectEncoder for logging.
func (tCtx *TransformContext) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddObject("resource", logging.Resource(tCtx.GetResource()))
	err = errors.Join(err, encoder.AddObject("scope", logging.InstrumentationScope(tCtx.GetInstrumentationScope())))
	err = errors.Join(err, encoder.AddObject("metric", logging.Metric(tCtx.metric)))

	switch dp := tCtx.dataPoint.(type) {
	case pmetric.NumberDataPoint:
		err = encoder.AddObject("datapoint", logging.NumberDataPoint(dp))
	case pmetric.HistogramDataPoint:
		err = encoder.AddObject("datapoint", logging.HistogramDataPoint(dp))
	case pmetric.ExponentialHistogramDataPoint:
		err = encoder.AddObject("datapoint", logging.ExponentialHistogramDataPoint(dp))
	case pmetric.SummaryDataPoint:
		err = encoder.AddObject("datapoint", logging.SummaryDataPoint(dp))
	}

	err = errors.Join(err, encoder.AddObject("cache", logging.Map(tCtx.cache)))
	return err
}

type TransformContextOption func(*TransformContext)

// NewTransformContextPtr returns a new TransformContext with the provided parameters from a pool of contexts.
// Caller must call TransformContext.Close on the returned TransformContext.
func NewTransformContextPtr(resourceMetrics pmetric.ResourceMetrics, scopeMetrics pmetric.ScopeMetrics, metric pmetric.Metric, dataPoint any, options ...TransformContextOption) *TransformContext {
	tCtx := tcPool.Get().(*TransformContext)
	tCtx.resourceMetrics = resourceMetrics
	tCtx.scopeMetrics = scopeMetrics
	tCtx.metric = metric
	tCtx.dataPoint = dataPoint
	for _, opt := range options {
		opt(tCtx)
	}
	return tCtx
}

// Close the current TransformContext.
// After this function returns this instance cannot be used.
func (tCtx *TransformContext) Close() {
	tCtx.resourceMetrics = pmetric.ResourceMetrics{}
	tCtx.scopeMetrics = pmetric.ScopeMetrics{}
	tCtx.metric = pmetric.Metric{}
	tCtx.dataPoint = nil
	tCtx.cache.Clear()
	tcPool.Put(tCtx)
}

// GetDataPoint returns the datapoint from the TransformContext.
func (tCtx *TransformContext) GetDataPoint() any {
	return tCtx.dataPoint
}

// GetInstrumentationScope returns the instrumentation scope from the TransformContext.
func (tCtx *TransformContext) GetInstrumentationScope() pcommon.InstrumentationScope {
	return tCtx.scopeMetrics.Scope()
}

// GetResource returns the resource from the TransformContext.
func (tCtx *TransformContext) GetResource() pcommon.Resource {
	return tCtx.resourceMetrics.Resource()
}

// GetMetric returns the metric from the TransformContext.
func (tCtx *TransformContext) GetMetric() pmetric.Metric {
	return tCtx.metric
}

// GetMetrics returns the metric slice from the TransformContext.
func (tCtx *TransformContext) GetMetrics() pmetric.MetricSlice {
	return tCtx.scopeMetrics.Metrics()
}

// GetScopeSchemaURLItem returns the scope schema URL item from the TransformContext.
func (tCtx *TransformContext) GetScopeSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.scopeMetrics
}

// GetResourceSchemaURLItem returns the resource schema URL item from the TransformContext.
func (tCtx *TransformContext) GetResourceSchemaURLItem() ctxcommon.SchemaURLItem {
	return tCtx.resourceMetrics
}

// EnablePathContextNames enables the support for path's context names on statements.
// When this option is configured, all statement's paths must have a valid context prefix,
// otherwise an error is reported.
//
// Experimental: *NOTE* this option is subject to change or removal in the future.
func EnablePathContextNames() ottl.Option[*TransformContext] {
	return func(p *ottl.Parser[*TransformContext]) {
		ottl.WithPathContextNames[*TransformContext]([]string{
			ctxdatapoint.Name,
			ctxresource.Name,
			ctxscope.LegacyName,
			ctxscope.Name,
			ctxmetric.Name,
		})(p)
	}
}

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

// NewParser creates a new Datapoint parser with the provided functions and options.
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
		if enum, ok := ctxdatapoint.SymbolTable[*val]; ok {
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
		ctxdatapoint.Name,
		ctxdatapoint.DocRef,
		cacheGetter,
		map[string]ottl.PathExpressionParser[*TransformContext]{
			ctxresource.Name:    ctxresource.PathGetSetter[*TransformContext],
			ctxscope.Name:       ctxscope.PathGetSetter[*TransformContext],
			ctxscope.LegacyName: ctxscope.PathGetSetter[*TransformContext],
			ctxmetric.Name:      ctxmetric.PathGetSetter[*TransformContext],
			ctxdatapoint.Name:   ctxdatapoint.PathGetSetter[*TransformContext],
		})
}
