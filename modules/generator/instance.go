package generator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/tempopb"
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

	registry *registry.ManagedRegistry
	wal      storage.Storage

	// processorsMtx protects the processors map, not the processors itself
	processorsMtx sync.RWMutex
	processors    map[string]processor.Processor

	shutdownCh chan struct{}

	reg    prometheus.Registerer
	logger log.Logger
}

func newInstance(cfg *Config, instanceID string, overrides metricsGeneratorOverrides, wal storage.Storage, reg prometheus.Registerer, logger log.Logger) (*instance, error) {
	logger = log.With(logger, "tenant", instanceID)

	i := &instance{
		cfg:        cfg,
		instanceID: instanceID,
		overrides:  overrides,

		registry: registry.New(&cfg.Registry, overrides, instanceID, wal, logger),
		wal:      wal,

		processors: make(map[string]processor.Processor),

		shutdownCh: make(chan struct{}, 1),

		reg:    reg,
		logger: logger,
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
				level.Error(i.logger).Log("msg", "updating the processors failed", "err", err)
			}

		case <-i.shutdownCh:
			return
		}
	}
}

func (i *instance) updateProcessors(desiredProcessors map[string]struct{}) error {
	i.processorsMtx.RLock()
	toAdd, toRemove := i.diffProcessors(desiredProcessors)
	i.processorsMtx.RUnlock()

	if len(toAdd) == 0 && len(toRemove) == 0 {
		return nil
	}

	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	for _, processorName := range toAdd {
		err := i.addProcessor(processorName)
		if err != nil {
			return err
		}
	}
	for _, processorName := range toRemove {
		i.removeProcessor(processorName)
	}

	i.updateProcessorMetrics()

	return nil
}

// diffProcessors compares the existings processors with desiredProcessors. Must be called under a
// read lock.
func (i *instance) diffProcessors(desiredProcessors map[string]struct{}) (toAdd []string, toRemove []string) {
	for processorName := range desiredProcessors {
		if _, ok := i.processors[processorName]; !ok {
			toAdd = append(toAdd, processorName)
		}
	}
	for processorName := range i.processors {
		if _, ok := desiredProcessors[processorName]; !ok {
			toRemove = append(toRemove, processorName)
		}
	}
	return toAdd, toRemove
}

// addProcessor registers the processor and adds it to the processors map. Must be called under a
// write lock.
func (i *instance) addProcessor(processorName string) error {
	level.Debug(i.logger).Log("msg", "adding processor", "processorName", processorName)

	var newProcessor processor.Processor
	switch processorName {
	case spanmetrics.Name:
		newProcessor = spanmetrics.New(i.cfg.Processor.SpanMetrics, i.registry)
	case servicegraphs.Name:
		newProcessor = servicegraphs.New(i.cfg.Processor.ServiceGraphs, i.instanceID, i.registry, i.logger)
	default:
		level.Error(i.logger).Log(
			"msg", fmt.Sprintf("processor does not exist, supported processors: [%s]", strings.Join(allSupportedProcessors, ", ")),
			"processorName", processorName,
		)
		return fmt.Errorf("unknown processor %s", processorName)
	}

	// check the processor wasn't added in the meantime
	if _, ok := i.processors[processorName]; ok {
		return nil
	}

	i.processors[processorName] = newProcessor

	return nil
}

// removeProcessor removes the processor from the processors map and shuts it down. Must be called
// under a write lock.
func (i *instance) removeProcessor(processorName string) {
	level.Debug(i.logger).Log("msg", "removing processor", "processorName", processorName)

	deletedProcessor, ok := i.processors[processorName]
	if !ok {
		return
	}

	delete(i.processors, processorName)

	deletedProcessor.Shutdown(context.Background())
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

func (i *instance) pushSpans(ctx context.Context, req *tempopb.PushSpansRequest) {
	i.updatePushMetrics(req)

	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	for _, processor := range i.processors {
		processor.PushSpans(ctx, req)
	}
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

// shutdown stops the instance and flushes any remaining data. After shutdown
// is called pushSpans should not be called anymore.
func (i *instance) shutdown() {
	close(i.shutdownCh)

	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	for processorName := range i.processors {
		i.removeProcessor(processorName)
	}

	i.registry.Close()

	err := i.wal.Close()
	if err != nil {
		level.Error(i.logger).Log("msg", "closing wal failed", "tenant", i.instanceID, "err", err)
	}
}
