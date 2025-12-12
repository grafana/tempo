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
	resourceFunctions                   map[string]ottl.Factory[ottlresource.TransformContext]
	dataPointFunctions                  map[string]ottl.Factory[ottldatapoint.TransformContext]
	logFunctions                        map[string]ottl.Factory[ottllog.TransformContext]
	metricFunctions                     map[string]ottl.Factory[ottlmetric.TransformContext]
	spanEventFunctions                  map[string]ottl.Factory[ottlspanevent.TransformContext]
	spanFunctions                       map[string]ottl.Factory[ottlspan.TransformContext]
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

// WithResourceFunctions will override the default OTTL resource context functions with the provided resourceFunctions in resulting processor.
// Subsequent uses of WithResourceFunctions will merge the provided resourceFunctions with the previously registered functions.
func WithResourceFunctions(resourceFunctions []ottl.Factory[ottlresource.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultResourceFunctionsOverridden {
			factory.resourceFunctions = map[string]ottl.Factory[ottlresource.TransformContext]{}
			factory.defaultResourceFunctionsOverridden = true
		}
		factory.resourceFunctions = mergeFunctionsToMap(factory.resourceFunctions, resourceFunctions)
	}
}

// WithDataPointFunctions will override the default OTTL datapoint context functions with the provided dataPointFunctions in resulting processor.
// Subsequent uses of WithDataPointFunctions will merge the provided dataPointFunctions with the previously registered functions.
func WithDataPointFunctions(dataPointFunctions []ottl.Factory[ottldatapoint.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultDataPointFunctionsOverridden {
			factory.dataPointFunctions = map[string]ottl.Factory[ottldatapoint.TransformContext]{}
			factory.defaultDataPointFunctionsOverridden = true
		}
		factory.dataPointFunctions = mergeFunctionsToMap(factory.dataPointFunctions, dataPointFunctions)
	}
}

// WithLogFunctions will override the default OTTL log context functions with the provided logFunctions in the resulting processor.
// Subsequent uses of WithLogFunctions will merge the provided logFunctions with the previously registered functions.
func WithLogFunctions(logFunctions []ottl.Factory[ottllog.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultLogFunctionsOverridden {
			factory.logFunctions = map[string]ottl.Factory[ottllog.TransformContext]{}
			factory.defaultLogFunctionsOverridden = true
		}
		factory.logFunctions = mergeFunctionsToMap(factory.logFunctions, logFunctions)
	}
}

// WithMetricFunctions will override the default OTTL metric context functions with the provided metricFunctions in the resulting processor.
// Subsequent uses of WithMetricFunctions will merge the provided metricFunctions with the previously registered functions.
func WithMetricFunctions(metricFunctions []ottl.Factory[ottlmetric.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultMetricFunctionsOverridden {
			factory.metricFunctions = map[string]ottl.Factory[ottlmetric.TransformContext]{}
			factory.defaultMetricFunctionsOverridden = true
		}
		factory.metricFunctions = mergeFunctionsToMap(factory.metricFunctions, metricFunctions)
	}
}

// WithSpanEventFunctions will override the default OTTL spanevent context functions with the provided spanEventFunctions in the resulting processor.
// Subsequent uses of WithSpanEventFunctions will merge the provided spanEventFunctions with the previously registered functions.
func WithSpanEventFunctions(spanEventFunctions []ottl.Factory[ottlspanevent.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultSpanEventFunctionsOverridden {
			factory.spanEventFunctions = map[string]ottl.Factory[ottlspanevent.TransformContext]{}
			factory.defaultSpanEventFunctionsOverridden = true
		}
		factory.spanEventFunctions = mergeFunctionsToMap(factory.spanEventFunctions, spanEventFunctions)
	}
}

// WithSpanFunctions will override the default OTTL span context functions with the provided spanFunctions in the resulting processor.
// Subsequent uses of WithSpanFunctions will merge the provided spanFunctions with the previously registered functions.
func WithSpanFunctions(spanFunctions []ottl.Factory[ottlspan.TransformContext]) FactoryOption {
	return func(factory *filterProcessorFactory) {
		if !factory.defaultSpanFunctionsOverridden {
			factory.spanFunctions = map[string]ottl.Factory[ottlspan.TransformContext]{}
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
