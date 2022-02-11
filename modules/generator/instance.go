package generator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/tracing"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/log"
)

var (
	allSupportedProcessors = []string{servicegraphs.Name, spanmetrics.Name}

	metricActiveProcessors = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_active_processors",
		Help:      "The active processors per tenant",
	}, []string{"tenant", "processor"})
	metricActiveProcessorsUpdateFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_active_processors_update_failed_total",
		Help:      "The total number of times updating the active processors failed",
	}, []string{"tenant"})
	metricSpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_received_total",
		Help:      "The total number of spans received per tenant",
	}, []string{"tenant"})
	metricBytesIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_bytes_received_total",
		Help:      "The total number of proto bytes received per tenant",
	}, []string{"tenant"})
)

type instance struct {
	cfg *Config

	instanceID string
	overrides  metricsGeneratorOverrides

	registry   processor.Registry
	appendable storage.Appendable

	// processorsMtx protects the processors map, not the processors itself
	processorsMtx sync.RWMutex
	processors    map[string]processor.Processor

	shutdownCh chan struct{}
}

func newInstance(cfg *Config, instanceID string, overrides metricsGeneratorOverrides, appendable storage.Appendable) (*instance, error) {
	i := &instance{
		cfg:        cfg,
		instanceID: instanceID,
		overrides:  overrides,

		registry:   processor.NewRegistry(cfg.ExternalLabels),
		appendable: appendable,

		processors: make(map[string]processor.Processor),

		shutdownCh: make(chan struct{}, 1),
	}

	err := i.updateProcessors(i.overrides.MetricsGeneratorProcessors(i.instanceID))
	if err != nil {
		return nil, fmt.Errorf("could not initialize processors: %w", err)
	}
	go i.watchOverrides()

	return i, nil
}

func (i *instance) watchOverrides() {
	reloadPeriod := 10 * time.Second

	ticker := time.NewTicker(reloadPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := i.updateProcessors(i.overrides.MetricsGeneratorProcessors(i.instanceID))
			if err != nil {
				metricActiveProcessorsUpdateFailed.WithLabelValues(i.instanceID).Inc()
				level.Error(log.Logger).Log("msg", "updating the processors failed", "err", err, "tenant", i.instanceID)
			}

		case <-i.shutdownCh:
			return
		}
	}
}

func (i *instance) updateProcessors(desiredProcessors map[string]struct{}) error {
	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	// add missing processors
	for processorName := range desiredProcessors {
		_, ok := i.processors[processorName]
		if ok {
			// processor already exists
			continue
		}

		var newProcessor processor.Processor
		switch processorName {
		case spanmetrics.Name:
			newProcessor = spanmetrics.New(i.cfg.Processor.SpanMetrics, i.instanceID)
		case servicegraphs.Name:
			newProcessor = servicegraphs.New(i.cfg.Processor.ServiceGraphs, i.instanceID)
		default:
			level.Error(log.Logger).Log(
				"msg", fmt.Sprintf("processor does not exist, supported processors: [%s]", strings.Join(allSupportedProcessors, ", ")),
				"processorName", processorName,
				"tenant", i.instanceID,
			)
			// this is most likely a misconfiguration, abort updateProcessors before we remove any active processors
			return fmt.Errorf("unknown processor %s", processorName)
		}

		i.processorsMtx.RUnlock()
		err := i.addProcessor(newProcessor)
		i.processorsMtx.RLock()

		if err != nil {
			return err
		}
	}

	// remove processors that are not in desiredProcessors
	for processorName := range i.processors {
		_, ok := desiredProcessors[processorName]
		if ok {
			// processor is also in the desired list
			continue
		}

		i.processorsMtx.RUnlock()
		i.removeProcessor(processorName)
		i.processorsMtx.RLock()
	}

	i.updateProcessorMetrics()

	return nil
}

func (i *instance) addProcessor(processor processor.Processor) error {
	level.Debug(log.Logger).Log("msg", "adding processor", "processorName", processor.Name(), "tenant", i.instanceID)

	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	// check the processor wasn't added in the meantime
	if _, ok := i.processors[processor.Name()]; ok {
		return nil
	}

	err := processor.RegisterMetrics(i.registry)
	if err != nil {
		return fmt.Errorf("error registering metrics for %s: %w", processor.Name(), err)
	}

	i.processors[processor.Name()] = processor

	return nil
}

func (i *instance) removeProcessor(processorName string) {
	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	deletedProcessor, ok := i.processors[processorName]
	if !ok {
		return
	}

	delete(i.processors, processorName)

	err := deletedProcessor.Shutdown(context.TODO(), i.registry)
	if err != nil {
		level.Error(log.Logger).Log("msg", "processor did not shutdown cleanly", "name", deletedProcessor.Name(), "err", err, "tenant", i.instanceID)
	}

	level.Debug(log.Logger).Log("msg", "removed processor", "processorName", processorName, "tenant", i.instanceID)
}

// updateProcessorMetrics updates the active processor metrics. Must be called under a read lock.
func (i *instance) updateProcessorMetrics() {
	for _, processorName := range allSupportedProcessors {
		isPresent := 0.0
		if _, ok := i.processors[processorName]; ok {
			isPresent = 1.0
		}
		metricActiveProcessors.WithLabelValues(i.instanceID, processorName).Set(isPresent)
	}
}

func (i *instance) pushSpans(ctx context.Context, req *tempopb.PushSpansRequest) error {
	i.updatePushMetrics(req)

	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	for _, processor := range i.processors {
		if err := processor.PushSpans(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) updatePushMetrics(req *tempopb.PushSpansRequest) {
	size := 0
	spanCount := 0
	for _, b := range req.Batches {
		size += b.Size()
		for _, ils := range b.InstrumentationLibrarySpans {
			spanCount += len(ils.Spans)
		}
	}
	metricBytesIngested.WithLabelValues(i.instanceID).Add(float64(size))
	metricSpansIngested.WithLabelValues(i.instanceID).Add(float64(spanCount))
}

func (i *instance) collectAndPushMetrics(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "instance.collectAndPushMetrics")
	defer span.Finish()

	traceID, _ := tracing.ExtractTraceID(ctx)
	level.Info(log.Logger).Log("msg", "collecting metrics", "tenant", i.instanceID, "traceID", traceID)

	appender := i.appendable.Appender(ctx)

	err := i.registry.Gather(appender)
	if err != nil {
		return err
	}

	return appender.Commit()
}

// shutdown stops the instance and flushes any remaining data. After shutdown
// is called PushSpans should not be called anymore.
func (i *instance) shutdown(ctx context.Context) error {
	close(i.shutdownCh)

	err := i.collectAndPushMetrics(ctx)
	if err != nil {
		level.Error(log.Logger).Log("msg", "collecting metrics failed at shutdown", "tenant", i.instanceID, "err", err)
	}

	i.processorsMtx.RLock()
	for processorName := range i.processors {
		i.processorsMtx.RUnlock()
		i.removeProcessor(processorName)
		i.processorsMtx.RLock()
	}
	i.processorsMtx.RUnlock()

	return err
}
