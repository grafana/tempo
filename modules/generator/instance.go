package generator

import (
	"context"
	"fmt"
	"reflect"
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
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
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
	metricSpansDiscarded = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_spans_discarded_total",
		Help:      "The total number of discarded spans received per tenant",
	}, []string{"tenant", "reason"})
)

const reasonOutsideTimeRangeSlack = "outside_metrics_ingestion_slack"

type instance struct {
	cfg *Config

	instanceID string
	overrides  metricsGeneratorOverrides

	registry *registry.ManagedRegistry
	wal      storage.Storage

	// processorsMtx protects the processors map, not the processors itself
	processorsMtx sync.RWMutex
	// processors is a map of processor name -> processor, only one instance of a processor can be
	// active at any time
	processors map[string]processor.Processor

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

	err := i.updateProcessors()
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
			err := i.updateProcessors()
			if err != nil {
				metricActiveProcessorsUpdateFailed.WithLabelValues(i.instanceID).Inc()
				level.Error(i.logger).Log("msg", "updating the processors failed", "err", err)
			}

		case <-i.shutdownCh:
			return
		}
	}
}

// Look at the processors defined and see if any are actually span-metrics subprocessors
// If they are, set the appropriate flags in the spanmetrics struct
func (i *instance) updateSubprocessors(desiredProcessors map[string]struct{}, desiredCfg ProcessorConfig) (map[string]struct{}, ProcessorConfig) {
	_, allOk := desiredProcessors["span-metrics"]
	_, countOk := desiredProcessors["span-metrics-count"]
	_, latencyOk := desiredProcessors["span-metrics-latency"]

	if allOk {
		delete(desiredProcessors, "span-metrics-latency")
		delete(desiredProcessors, "span-metrics-count")
	} else if countOk && latencyOk {
		delete(desiredProcessors, "span-metrics-latency")
		delete(desiredProcessors, "span-metrics-count")
		desiredProcessors["span-metrics"] = struct{}{}
		desiredCfg.SpanMetrics.Subprocessors["Latency"] = true
		desiredCfg.SpanMetrics.Subprocessors["Count"] = true
	} else if countOk {
		delete(desiredProcessors, "span-metrics-count")
		desiredProcessors["span-metrics"] = struct{}{}
		desiredCfg.SpanMetrics.Subprocessors["Latency"] = false
		desiredCfg.SpanMetrics.Subprocessors["Count"] = true
		desiredCfg.SpanMetrics.HistogramBuckets = nil
	} else if latencyOk {
		delete(desiredProcessors, "span-metrics-latency")
		desiredProcessors["span-metrics"] = struct{}{}
		desiredCfg.SpanMetrics.Subprocessors["Count"] = false
		desiredCfg.SpanMetrics.Subprocessors["Latency"] = true
	}

	return desiredProcessors, desiredCfg
}

func (i *instance) updateProcessors() error {
	desiredProcessors := i.overrides.MetricsGeneratorProcessors(i.instanceID)
	desiredCfg, err := i.cfg.Processor.copyWithOverrides(i.overrides, i.instanceID)
	if err != nil {
		return err
	}

	desiredProcessors, desiredCfg = i.updateSubprocessors(desiredProcessors, desiredCfg)

	i.processorsMtx.RLock()
	toAdd, toRemove, toReplace, err := i.diffProcessors(desiredProcessors, desiredCfg)
	i.processorsMtx.RUnlock()

	if err != nil {
		return err
	}
	if len(toAdd) == 0 && len(toRemove) == 0 && len(toReplace) == 0 {
		return nil
	}

	i.processorsMtx.Lock()
	defer i.processorsMtx.Unlock()

	for _, processorName := range toAdd {
		err := i.addProcessor(processorName, desiredCfg)
		if err != nil {
			return err
		}
	}
	for _, processorName := range toRemove {
		i.removeProcessor(processorName)
	}
	for _, processorName := range toReplace {
		i.removeProcessor(processorName)

		err := i.addProcessor(processorName, desiredCfg)
		if err != nil {
			return err
		}
	}

	i.updateProcessorMetrics()

	return nil
}

// diffProcessors compares the existing processors with the desired processors and config.
// Must be called under a read lock.
func (i *instance) diffProcessors(desiredProcessors map[string]struct{}, desiredCfg ProcessorConfig) (toAdd, toRemove, toReplace []string, err error) {
	for processorName := range desiredProcessors {
		if _, ok := i.processors[processorName]; !ok {
			toAdd = append(toAdd, processorName)
		}
	}
	for processorName, processor := range i.processors {
		if _, ok := desiredProcessors[processorName]; !ok {
			toRemove = append(toRemove, processorName)
			continue
		}

		switch p := processor.(type) {
		case *spanmetrics.Processor:
			if !reflect.DeepEqual(p.Cfg, desiredCfg.SpanMetrics) {
				toReplace = append(toReplace, processorName)
			}
		case *servicegraphs.Processor:
			if !reflect.DeepEqual(p.Cfg, desiredCfg.ServiceGraphs) {
				toReplace = append(toReplace, processorName)
			}
		default:
			level.Error(i.logger).Log(
				"msg", fmt.Sprintf("processor does not exist, supported processors: [%s]", strings.Join(allSupportedProcessors, ", ")),
				"processorName", processorName,
			)
			err = fmt.Errorf("unknown processor %s", processorName)
			return
		}
	}
	return
}

// addProcessor registers the processor and adds it to the processors map. Must be called under a
// write lock.
func (i *instance) addProcessor(processorName string, cfg ProcessorConfig) error {
	level.Debug(i.logger).Log("msg", "adding processor", "processorName", processorName)

	var newProcessor processor.Processor
	switch processorName {
	case spanmetrics.Name:
		newProcessor = spanmetrics.New(cfg.SpanMetrics, i.registry)
	case servicegraphs.Name:
		newProcessor = servicegraphs.New(cfg.ServiceGraphs, i.instanceID, i.registry, i.logger)
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
	i.preprocessSpans(req)
	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	for _, processor := range i.processors {
		processor.PushSpans(ctx, req)
	}
}

func (i *instance) preprocessSpans(req *tempopb.PushSpansRequest) {
	size := 0
	spanCount := 0
	expiredSpanCount := 0
	for _, b := range req.Batches {
		size += b.Size()
		for _, ss := range b.ScopeSpans {
			spanCount += len(ss.Spans)
			// filter spans that have end time > max_age and end time more than 5 days in the future
			newSpansArr := make([]*v1.Span, len(ss.Spans))
			timeNow := time.Now()
			index := 0
			for _, span := range ss.Spans {
				if span.EndTimeUnixNano >= uint64(timeNow.Add(-i.cfg.MetricsIngestionSlack).UnixNano()) && span.EndTimeUnixNano <= uint64(timeNow.Add(i.cfg.MetricsIngestionSlack).UnixNano()) {
					newSpansArr[index] = span
					index++
				} else {
					expiredSpanCount++
				}
			}
			ss.Spans = newSpansArr[0:index]
		}
	}
	i.updatePushMetrics(size, spanCount, expiredSpanCount)
}

func (i *instance) updatePushMetrics(bytesIngested int, spanCount int, expiredSpanCount int) {
	metricBytesIngested.WithLabelValues(i.instanceID).Add(float64(bytesIngested))
	metricSpansIngested.WithLabelValues(i.instanceID).Add(float64(spanCount))
	metricSpansDiscarded.WithLabelValues(i.instanceID, reasonOutsideTimeRangeSlack).Add(float64(expiredSpanCount))
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
