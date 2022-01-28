package generator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/log/level"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/storage"
	"github.com/weaveworks/common/tracing"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/tempopb"
)

var (
	allSupportedProcessors = []string{servicegraphs.Name, spanmetrics.Name}

	metricActiveProcessors = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_active_processors",
		Help:      "The active processors per tenant",
	}, []string{"tenant", "processor"})
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
	overrides  *overrides.Overrides

	registry   processor.Registry
	appendable storage.Appendable

	// processorsMtx protects the processors map, not the processors itself
	processorsMtx sync.RWMutex
	processors    map[string]processor.Processor

	shutdownCh chan struct{}

	metricSpansIngestedTotal prometheus.Counter
	metricBytesIngestedTotal prometheus.Counter
}

func newInstance(cfg *Config, instanceID string, overrides *overrides.Overrides, appendable storage.Appendable) (*instance, error) {
	i := &instance{
		cfg:        cfg,
		instanceID: instanceID,
		overrides:  overrides,

		registry:   processor.NewRegistry(cfg.ExternalLabels),
		appendable: appendable,

		processors: make(map[string]processor.Processor),

		shutdownCh: make(chan struct{}, 1),

		metricSpansIngestedTotal: metricSpansIngested.WithLabelValues(instanceID),
		metricBytesIngestedTotal: metricBytesIngested.WithLabelValues(instanceID),
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
				level.Error(log.Logger).Log("msg", "updating the processors failed", "err", err, "tenant", i.instanceID)
			}

		case <-i.shutdownCh:
			return
		}
	}
}

func (i *instance) updateProcessors(desiredProcessors map[string]struct{}) error {
	// add missing processors
	for processorName := range desiredProcessors {
		i.processorsMtx.RLock()
		_, ok := i.processors[processorName]
		i.processorsMtx.RUnlock()

		if ok {
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

		err := i.addProcessor(newProcessor)
		if err != nil {
			return err
		}
	}

	// remove processors that are not in desiredProcessors
	for processorName := range i.processors {
		_, ok := desiredProcessors[processorName]
		if ok {
			continue
		}

		i.removeProcessor(processorName)
	}

	i.updateProcessorMetrics()

	return nil
}

func (i *instance) updateProcessorMetrics() {
	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	for _, processorName := range allSupportedProcessors {
		isPresent := 0.0
		if _, ok := i.processors[processorName]; ok {
			isPresent = 1.0
		}
		metricActiveProcessors.WithLabelValues(i.instanceID, processorName).Set(isPresent)
	}
}

func (i *instance) addProcessor(processor processor.Processor) error {
	level.Debug(log.Logger).Log("msg", "adding processor", "processorName", processor.Name(), "tenant", i.instanceID)

	err := processor.RegisterMetrics(i.registry)
	if err != nil {
		return fmt.Errorf("error registering metrics for %s: %w", processor.Name(), err)
	}

	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	i.processors[processor.Name()] = processor

	return nil
}

func (i *instance) removeProcessor(processorName string) {
	i.processorsMtx.Lock()
	deletedProcessor := i.processors[processorName]
	delete(i.processors, processorName)
	i.processorsMtx.Unlock()

	err := deletedProcessor.Shutdown(context.TODO(), i.registry)
	if err != nil {
		level.Error(log.Logger).Log("msg", "processor did not shutdown cleanly", "name", deletedProcessor.Name(), "err", err, "tenant", i.instanceID)
	}

	level.Debug(log.Logger).Log("msg", "removed processor", "processorName", processorName, "tenant", i.instanceID)
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
	i.metricBytesIngestedTotal.Add(float64(size))
	i.metricSpansIngestedTotal.Add(float64(spanCount))
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
	// TODO should we set a boolean to refuse push request once this is called?

	i.shutdownCh <- struct{}{}

	err := i.collectAndPushMetrics(ctx)
	if err != nil {
		level.Error(log.Logger).Log("msg", "collecting metrics failed at shutdown", "tenant", i.instanceID, "err", err)
	}

	for processorName := range i.processors {
		i.removeProcessor(processorName)
	}

	return nil
}
