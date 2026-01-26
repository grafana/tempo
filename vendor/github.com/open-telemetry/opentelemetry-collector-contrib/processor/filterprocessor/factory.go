// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlprofile"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor/internal/metadata"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

type filterProcessorFactory struct {
	resourceFunctions                   map[string]ottl.Factory[*ottlresource.TransformContext]
	dataPointFunctions                  map[string]ottl.Factory[*ottldatapoint.TransformContext]
	logFunctions                        map[string]ottl.Factory[*ottllog.TransformContext]
	metricFunctions                     map[string]ottl.Factory[*ottlmetric.TransformContext]
	spanEventFunctions                  map[string]ottl.Factory[*ottlspanevent.TransformContext]
	spanFunctions                       map[string]ottl.Factory[*ottlspan.TransformContext]
	profileFunctions                    map[string]ottl.Factory[ottlprofile.TransformContext]
	defaultResourceFunctionsOverridden  bool
	defaultDataPointFunctionsOverridden bool
	defaultLogFunctionsOverridden       bool
	defaultMetricFunctionsOverridden    bool
	defaultSpanEventFunctionsOverridden bool
	defaultSpanFunctionsOverridden      bool
	defaultProfileFunctionsOverridden   bool
}

// FactoryOption applies changes to filterProcessorFactory.
type FactoryOption func(factory *filterProcessorFactory)

// Deprecated: [v0.142.0] Use WithResourceFunctionsNew.
func WithResourceFunctions(resourceFunctions []ottl.Factory[ottlresource.TransformContext]) FactoryOption {
	newResourceFunctions := make([]ottl.Factory[*ottlresource.TransformContext], 0, len(resourceFunctions))
	for _, resourceFunction := range resourceFunctions {
		newResourceFunctions = append(newResourceFunctions, ottl.NewFactory[*ottlresource.TransformContext](resourceFunction.Name(), resourceFunction.CreateDefaultArguments(), fromNonPointerFunction(resourceFunction.CreateFunction)))
	}
	return WithResourceFunctionsNew(newResourceFunctions)
}

// WithResourceFunctionsNew will override the default OTTL resource context functions with the provided resourceFunctions in resulting processor.
// Subsequent uses of WithResourceFunctionsNew will merge the provided resourceFunctions with the previously registered functions.
func WithResourceFunctionsNew(resourceFunctions []ottl.Factory[*ottlresource.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultResourceFunctionsOverridden {
			factory.resourceFunctions = map[string]ottl.Factory[*ottlresource.TransformContext]{}
			factory.defaultResourceFunctionsOverridden = true
		}
		factory.resourceFunctions = mergeFunctionsToMap(factory.resourceFunctions, resourceFunctions)
	}
}

// Deprecated: [v0.142.0] Use WithDataPointFunctionsNew.
func WithDataPointFunctions(dataPointFunctions []ottl.Factory[ottldatapoint.TransformContext]) FactoryOption {
	newDataPointFunctions := make([]ottl.Factory[*ottldatapoint.TransformContext], 0, len(dataPointFunctions))
	for _, dataPointFunction := range dataPointFunctions {
		newDataPointFunctions = append(newDataPointFunctions, ottl.NewFactory[*ottldatapoint.TransformContext](dataPointFunction.Name(), dataPointFunction.CreateDefaultArguments(), fromNonPointerFunction(dataPointFunction.CreateFunction)))
	}
	return WithDataPointFunctionsNew(newDataPointFunctions)
}

// WithDataPointFunctionsNew will override the default OTTL datapoint context functions with the provided dataPointFunctions in resulting processor.
// Subsequent uses of WithDataPointFunctionsNew will merge the provided dataPointFunctions with the previously registered functions.
func WithDataPointFunctionsNew(dataPointFunctions []ottl.Factory[*ottldatapoint.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultDataPointFunctionsOverridden {
			factory.dataPointFunctions = map[string]ottl.Factory[*ottldatapoint.TransformContext]{}
			factory.defaultDataPointFunctionsOverridden = true
		}
		factory.dataPointFunctions = mergeFunctionsToMap(factory.dataPointFunctions, dataPointFunctions)
	}
}

// Deprecated: [v0.142.0] Use WithLogFunctionsNew.
func WithLogFunctions(logFunctions []ottl.Factory[ottllog.TransformContext]) FactoryOption {
	newLogFunctions := make([]ottl.Factory[*ottllog.TransformContext], 0, len(logFunctions))
	for _, logFunction := range logFunctions {
		newLogFunctions = append(newLogFunctions, ottl.NewFactory[*ottllog.TransformContext](logFunction.Name(), logFunction.CreateDefaultArguments(), fromNonPointerFunction(logFunction.CreateFunction)))
	}
	return WithLogFunctionsNew(newLogFunctions)
}

// WithLogFunctionsNew will override the default OTTL log context functions with the provided logFunctions in the resulting processor.
// Subsequent uses of WithLogFunctionsNew will merge the provided logFunctions with the previously registered functions.
func WithLogFunctionsNew(logFunctions []ottl.Factory[*ottllog.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultLogFunctionsOverridden {
			factory.logFunctions = map[string]ottl.Factory[*ottllog.TransformContext]{}
			factory.defaultLogFunctionsOverridden = true
		}
		factory.logFunctions = mergeFunctionsToMap(factory.logFunctions, logFunctions)
	}
}

// Deprecated: [v0.142.0] Use WithMetricFunctionsNew.
func WithMetricFunctions(metricFunctions []ottl.Factory[ottlmetric.TransformContext]) FactoryOption {
	newMetricFunctions := make([]ottl.Factory[*ottlmetric.TransformContext], 0, len(metricFunctions))
	for _, metricFunction := range metricFunctions {
		newMetricFunctions = append(newMetricFunctions, ottl.NewFactory[*ottlmetric.TransformContext](metricFunction.Name(), metricFunction.CreateDefaultArguments(), fromNonPointerFunction(metricFunction.CreateFunction)))
	}
	return WithMetricFunctionsNew(newMetricFunctions)
}

// WithMetricFunctionsNew will override the default OTTL metric context functions with the provided metricFunctions in the resulting processor.
// Subsequent uses of WithMetricFunctionsNew will merge the provided metricFunctions with the previously registered functions.
func WithMetricFunctionsNew(metricFunctions []ottl.Factory[*ottlmetric.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultMetricFunctionsOverridden {
			factory.metricFunctions = map[string]ottl.Factory[*ottlmetric.TransformContext]{}
			factory.defaultMetricFunctionsOverridden = true
		}
		factory.metricFunctions = mergeFunctionsToMap(factory.metricFunctions, metricFunctions)
	}
}

// Deprecated: [v0.142.0] Use WithSpanEventFunctionsNew.
func WithSpanEventFunctions(spanEventFunctions []ottl.Factory[ottlspanevent.TransformContext]) FactoryOption {
	newSpanFunctions := make([]ottl.Factory[*ottlspanevent.TransformContext], 0, len(spanEventFunctions))
	for _, spanEventFunction := range spanEventFunctions {
		newSpanFunctions = append(newSpanFunctions, ottl.NewFactory[*ottlspanevent.TransformContext](spanEventFunction.Name(), spanEventFunction.CreateDefaultArguments(), fromNonPointerFunction(spanEventFunction.CreateFunction)))
	}
	return WithSpanEventFunctionsNew(newSpanFunctions)
}

// WithSpanEventFunctionsNew will override the default OTTL spanevent context functions with the provided spanEventFunctions in the resulting processor.
// Subsequent uses of WithSpanEventFunctionsNew will merge the provided spanEventFunctions with the previously registered functions.
func WithSpanEventFunctionsNew(spanEventFunctions []ottl.Factory[*ottlspanevent.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultSpanEventFunctionsOverridden {
			factory.spanEventFunctions = map[string]ottl.Factory[*ottlspanevent.TransformContext]{}
			factory.defaultSpanEventFunctionsOverridden = true
		}
		factory.spanEventFunctions = mergeFunctionsToMap(factory.spanEventFunctions, spanEventFunctions)
	}
}

// Deprecated: [v0.142.0] use WithSpanFunctionsNew.
func WithSpanFunctions(spanFunctions []ottl.Factory[ottlspan.TransformContext]) FactoryOption {
	newSpanFunctions := make([]ottl.Factory[*ottlspan.TransformContext], 0, len(spanFunctions))
	for _, spanFunction := range spanFunctions {
		newSpanFunctions = append(newSpanFunctions, ottl.NewFactory[*ottlspan.TransformContext](spanFunction.Name(), spanFunction.CreateDefaultArguments(), fromNonPointerFunction(spanFunction.CreateFunction)))
	}
	return WithSpanFunctionsNew(newSpanFunctions)
}

// WithSpanFunctionsNew will override the default OTTL span context functions with the provided spanFunctions in the resulting processor.
// Subsequent uses of WithSpanFunctions will merge the provided spanFunctions with the previously registered functions.
func WithSpanFunctionsNew(spanFunctions []ottl.Factory[*ottlspan.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultSpanFunctionsOverridden {
			factory.spanFunctions = map[string]ottl.Factory[*ottlspan.TransformContext]{}
			factory.defaultSpanFunctionsOverridden = true
		}
		factory.spanFunctions = mergeFunctionsToMap(factory.spanFunctions, spanFunctions)
	}
}

// WithProfileFunctions will override the default OTTL profile context functions with the provided profileFunctions in the resulting processor.
// Subsequent uses of WithProfileFunctions will merge the provided profileFunctions with the previously registered functions.
func WithProfileFunctions(profileFunctions []ottl.Factory[ottlprofile.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultProfileFunctionsOverridden {
			factory.profileFunctions = map[string]ottl.Factory[ottlprofile.TransformContext]{}
			factory.defaultProfileFunctionsOverridden = true
		}
		factory.profileFunctions = mergeFunctionsToMap(factory.profileFunctions, profileFunctions)
	}
}

// NewFactory returns a new factory for the Filter processor.
func NewFactory() processor.Factory {
	return NewFactoryWithOptions()
}

// NewFactoryWithOptions can receive FactoryOption like With*Functions to register non-default OTTL functions in the resulting processor.
func NewFactoryWithOptions(options ...FactoryOption) processor.Factory {
	f := &filterProcessorFactory{
		resourceFunctions:  defaultResourceFunctionsMap(),
		dataPointFunctions: defaultDataPointFunctionsMap(),
		logFunctions:       defaultLogFunctionsMap(),
		metricFunctions:    defaultMetricFunctionsMap(),
		spanEventFunctions: defaultSpanEventFunctionsMap(),
		spanFunctions:      defaultSpanFunctionsMap(),
		profileFunctions:   defaultProfileFunctionsMap(),
	}
	for _, o := range options {
		o(f)
	}

	return xprocessor.NewFactory(
		metadata.Type,
		f.createDefaultConfig,
		xprocessor.WithLogs(f.createLogsProcessor, metadata.LogsStability),
		xprocessor.WithTraces(f.createTracesProcessor, metadata.TracesStability),
		xprocessor.WithMetrics(f.createMetricsProcessor, metadata.MetricsStability),
		xprocessor.WithProfiles(f.createProfilesProcessor, metadata.ProfilesStability),
	)
}

func (f *filterProcessorFactory) createDefaultConfig() component.Config {
	return &Config{
		ErrorMode:          ottl.PropagateError,
		resourceFunctions:  f.resourceFunctions,
		dataPointFunctions: f.dataPointFunctions,
		logFunctions:       f.logFunctions,
		metricFunctions:    f.metricFunctions,
		spanEventFunctions: f.spanEventFunctions,
		spanFunctions:      f.spanFunctions,
		profileFunctions:   f.profileFunctions,
	}
}

func (f *filterProcessorFactory) createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	if f.defaultResourceFunctionsOverridden || f.defaultDataPointFunctionsOverridden || f.defaultMetricFunctionsOverridden {
		set.Logger.Debug("non-default OTTL metric functions have been registered in the \"filter\" processor",
			zap.Bool("resource", f.defaultResourceFunctionsOverridden),
			zap.Bool("metric", f.defaultMetricFunctionsOverridden),
			zap.Bool("datapoint", f.defaultDataPointFunctionsOverridden),
		)
	}
	fp, err := newFilterMetricProcessor(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetrics(
		ctx,
		set,
		cfg,
		nextConsumer,
		fp.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities))
}

func (f *filterProcessorFactory) createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	if f.defaultResourceFunctionsOverridden || f.defaultLogFunctionsOverridden {
		set.Logger.Debug("non-default OTTL log functions have been registered in the \"filter\" processor",
			zap.Bool("resource", f.defaultResourceFunctionsOverridden),
			zap.Bool("log", f.defaultLogFunctionsOverridden),
		)
	}
	fp, err := newFilterLogsProcessor(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		fp.processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}

func (f *filterProcessorFactory) createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	if f.defaultResourceFunctionsOverridden || f.defaultSpanEventFunctionsOverridden || f.defaultSpanFunctionsOverridden {
		set.Logger.Debug("non-default OTTL trace functions have been registered in the \"filter\" processor",
			zap.Bool("resource", f.defaultResourceFunctionsOverridden),
			zap.Bool("span", f.defaultSpanFunctionsOverridden),
			zap.Bool("spanevent", f.defaultSpanEventFunctionsOverridden),
		)
	}
	fp, err := newFilterSpansProcessor(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		fp.processTraces,
		processorhelper.WithCapabilities(processorCapabilities))
}

func (f *filterProcessorFactory) createProfilesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	if f.defaultResourceFunctionsOverridden || f.defaultProfileFunctionsOverridden {
		set.Logger.Debug("non-default OTTL profile functions have been registered in the \"filter\" processor",
			zap.Bool("resource", f.defaultResourceFunctionsOverridden),
			zap.Bool("profile", f.defaultProfileFunctionsOverridden),
		)
	}
	fp, err := newFilterProfilesProcessor(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		nextConsumer,
		fp.processProfiles,
		xprocessorhelper.WithCapabilities(processorCapabilities))
}

func fromNonPointerFunction[K any](legacy func(fCtx ottl.FunctionContext, args ottl.Arguments) (ottl.ExprFunc[K], error)) func(fCtx ottl.FunctionContext, args ottl.Arguments) (ottl.ExprFunc[*K], error) {
	return func(fCtx ottl.FunctionContext, args ottl.Arguments) (ottl.ExprFunc[*K], error) {
		legacyExpr, err := legacy(fCtx, args)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, tCtx *K) (any, error) {
			return legacyExpr(ctx, *tCtx)
		}, nil
	}
}
