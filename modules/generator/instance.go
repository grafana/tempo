package generator

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/tempodb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/modules/generator/processor/hostinfo"
	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/wal"

	"go.uber.org/atomic"
)

var (
	SupportedProcessors = []string{
		servicegraphs.Name,
		spanmetrics.Name,
		localblocks.Name,
		spanmetrics.Count.String(),
		spanmetrics.Latency.String(),
		spanmetrics.Size.String(),
	}

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
	metricSkippedProcessorPushes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "tempo",
		Name:      "metrics_generator_metrics_generation_skipped_processor_pushes_total",
		Help: "The total number of processor pushes skipped because the request indicated that" +
			" metrics should not be generated.",
	}, []string{"tenant"})
)

const (
	reasonOutsideTimeRangeSlack = "outside_metrics_ingestion_slack"
	reasonSpanMetricsFiltered   = "span_metrics_filtered"
	reasonInvalidUTF8           = "invalid_utf8"
)

type instance struct {
	cfg *Config

	instanceID             string
	overrides              metricsGeneratorOverrides
	ingestionSlackOverride atomic.Int64

	registry *registry.ManagedRegistry
	wal      storage.Storage

	traceWAL      *wal.WAL
	traceQueryWAL *wal.WAL
	writer        tempodb.Writer

	// processorsMtx protects the processors map, not the processors itself
	processorsMtx sync.RWMutex
	// processors is a map of processor name -> processor, only one instance of a processor can be
	// active at any time
	processors            map[string]processor.Processor
	queuebasedLocalBlocks *localblocks.Processor

	shutdownCh chan struct{}

	logger log.Logger
}

func newInstance(cfg *Config, instanceID string, overrides metricsGeneratorOverrides, wal storage.Storage, logger log.Logger, traceWAL, rf1TraceWAL *wal.WAL, writer tempodb.Writer) (*instance, error) {
	logger = log.With(logger, "tenant", instanceID)

	i := &instance{
		cfg:        cfg,
		instanceID: instanceID,
		overrides:  overrides,

		registry:      registry.New(&cfg.Registry, overrides, instanceID, wal, logger),
		wal:           wal,
		traceWAL:      traceWAL,
		traceQueryWAL: rf1TraceWAL,
		writer:        writer,

		processors: make(map[string]processor.Processor),

		shutdownCh: make(chan struct{}, 1),

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
	desiredProcessorsFound := false
	for d := range desiredProcessors {
		if (d == spanmetrics.Name) || (spanmetrics.ParseSubprocessor(d)) {
			desiredProcessorsFound = true
		}
	}

	if !desiredProcessorsFound {
		return desiredProcessors, desiredCfg
	}

	_, allOk := desiredProcessors[spanmetrics.Name]
	_, countOk := desiredProcessors[spanmetrics.Count.String()]
	_, latencyOk := desiredProcessors[spanmetrics.Latency.String()]
	_, sizeOk := desiredProcessors[spanmetrics.Size.String()]

	// Copy the map before modifying it. This map can be shared by multiple instances and is not safe to write to.
	newDesiredProcessors := map[string]struct{}{}
	maps.Copy(newDesiredProcessors, desiredProcessors)

	if !allOk {
		newDesiredProcessors[spanmetrics.Name] = struct{}{}
		desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Count] = false
		desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Latency] = false
		desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Size] = false
		desiredCfg.SpanMetrics.HistogramBuckets = nil

		if countOk {
			desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Count] = true
		}
		if latencyOk {
			desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Latency] = true
			desiredCfg.SpanMetrics.HistogramBuckets = prometheus.ExponentialBuckets(0.002, 2, 14)
		}
		if sizeOk {
			desiredCfg.SpanMetrics.Subprocessors[spanmetrics.Size] = true
		}
	}

	delete(newDesiredProcessors, spanmetrics.Latency.String())
	delete(newDesiredProcessors, spanmetrics.Count.String())
	delete(newDesiredProcessors, spanmetrics.Size.String())

	return newDesiredProcessors, desiredCfg
}

func (i *instance) updateProcessors() error {
	desiredProcessors := i.filterDisabledProcessors(i.overrides.MetricsGeneratorProcessors(i.instanceID))
	desiredCfg, err := i.cfg.Processor.copyWithOverrides(i.overrides, i.instanceID)
	if err != nil {
		return err
	}

	ingestionSlackInt := i.overrides.MetricsGeneratorIngestionSlack(i.instanceID).Nanoseconds()
	if ingestionSlackInt == 0 {
		ingestionSlackInt = i.cfg.MetricsIngestionSlack.Nanoseconds()
	}

	i.ingestionSlackOverride.Store(ingestionSlackInt)

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

// filterDisabledProcessors removes processors that should never be instantiated
// according to the generator's configuration from the given set of processors.
func (i *instance) filterDisabledProcessors(processors map[string]struct{}) map[string]struct{} {
	// If no processors are disabled, do not apply any filtering.
	if !i.cfg.DisableLocalBlocks {
		return processors
	}

	// Otherwise, do not instantiate the localblocks processor.
	filteredProcessors := maps.Clone(processors)
	delete(filteredProcessors, localblocks.Name)

	return filteredProcessors
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
		case *localblocks.Processor:
			if !reflect.DeepEqual(p.Cfg, desiredCfg.LocalBlocks) {
				toReplace = append(toReplace, processorName)
			}
		case *hostinfo.Processor:
			if !reflect.DeepEqual(p.Cfg, desiredCfg.HostInfo) {
				toReplace = append(toReplace, processorName)
			}
		default:
			level.Error(i.logger).Log(
				"msg", fmt.Sprintf("processor does not exist, supported processors: [%s]", strings.Join(SupportedProcessors, ", ")),
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
	var err error
	switch processorName {
	case spanmetrics.Name:
		filteredSpansCounter := metricSpansDiscarded.WithLabelValues(i.instanceID, reasonSpanMetricsFiltered)
		invalidUTF8Counter := metricSpansDiscarded.WithLabelValues(i.instanceID, reasonInvalidUTF8)
		newProcessor, err = spanmetrics.New(cfg.SpanMetrics, i.registry, filteredSpansCounter, invalidUTF8Counter)
		if err != nil {
			return err
		}
	case servicegraphs.Name:
		newProcessor = servicegraphs.New(cfg.ServiceGraphs, i.instanceID, i.registry, i.logger)
	case localblocks.Name:
		p, err := localblocks.New(cfg.LocalBlocks, i.instanceID, i.traceWAL, i.writer, i.overrides)
		if err != nil {
			return err
		}
		newProcessor = p

		// Add the non-flushing alternate if configured
		if i.traceQueryWAL != nil {
			nonFlushingConfig := cfg.LocalBlocks
			nonFlushingConfig.FlushToStorage = false
			nonFlushingConfig.AssertMaxLiveTraces = true
			nonFlushingConfig.AdjustTimeRangeForSlack = false
			i.queuebasedLocalBlocks, err = localblocks.New(nonFlushingConfig, i.instanceID, i.traceQueryWAL, i.writer, i.overrides)
			if err != nil {
				return err
			}
		}
	case hostinfo.Name:
		newProcessor, err = hostinfo.New(cfg.HostInfo, i.registry, i.logger)
		if err != nil {
			return err
		}
	default:
		level.Error(i.logger).Log(
			"msg", fmt.Sprintf("processor does not exist, supported processors: [%s]", strings.Join(SupportedProcessors, ", ")),
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

	if processorName == localblocks.Name && i.queuebasedLocalBlocks != nil {
		i.queuebasedLocalBlocks.Shutdown(context.Background())
		i.queuebasedLocalBlocks = nil
	}
}

// updateProcessorMetrics updates the active processor metrics. Must be called under a read lock.
func (i *instance) updateProcessorMetrics() {
	for _, processorName := range SupportedProcessors {
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
		switch processor.Name() {
		case localblocks.Name:
			processor.PushSpans(ctx, req)
		case spanmetrics.Name, servicegraphs.Name:
			if req.SkipMetricsGeneration {
				metricSkippedProcessorPushes.WithLabelValues(i.instanceID).Inc()
				break
			}
			processor.PushSpans(ctx, req)
		}
	}
}

func (i *instance) pushSpansFromQueue(ctx context.Context, ts time.Time, req *tempopb.PushSpansRequest) {
	i.preprocessSpans(req)
	i.processorsMtx.RLock()
	defer i.processorsMtx.RUnlock()

	for _, processor := range i.processors {
		switch processor.Name() {
		case localblocks.Name:
			// don't push to this processor as queue consumer, instead use queue based local
			// blocks if configured.
		case spanmetrics.Name, servicegraphs.Name:
			if req.SkipMetricsGeneration {
				metricSkippedProcessorPushes.WithLabelValues(i.instanceID).Inc()
				break
			}
			processor.PushSpans(ctx, req)
		}
	}

	// Now we push to the non-flushing local blocks if present
	if i.queuebasedLocalBlocks != nil {
		i.queuebasedLocalBlocks.DeterministicPush(ts, req)
	}
}

func (i *instance) preprocessSpans(req *tempopb.PushSpansRequest) {
	// TODO - uniqify all strings?
	// Doesn't help allocs, but should greatly reduce inuse space
	size := 0
	spanCount := 0
	expiredSpanCount := 0
	ingestionSlackNano := i.ingestionSlackOverride.Load()

	for _, b := range req.Batches {
		size += b.Size()
		for _, ss := range b.ScopeSpans {
			spanCount += len(ss.Spans)
			// filter spans that have end time > max_age and end time more than 5 days in the future
			newSpansArr := make([]*v1.Span, len(ss.Spans))
			timeNow := time.Now()
			maxTimePast := uint64(timeNow.UnixNano() - ingestionSlackNano)
			maxTimeFuture := uint64(timeNow.UnixNano() + ingestionSlackNano)

			index := 0
			for _, span := range ss.Spans {
				if span.EndTimeUnixNano >= maxTimePast && span.EndTimeUnixNano <= maxTimeFuture {
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

func (i *instance) GetMetrics(ctx context.Context, req *tempopb.SpanMetricsRequest) (resp *tempopb.SpanMetricsResponse, err error) {
	for _, processor := range i.processors {
		switch p := processor.(type) {
		case *localblocks.Processor:
			return p.GetMetrics(ctx, req)
		default:
		}
	}

	return nil, fmt.Errorf("localblocks processor not found")
}

func (i *instance) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (resp *tempopb.QueryRangeResponse, err error) {
	var processors []*localblocks.Processor

	i.processorsMtx.RLock()
	for _, processor := range i.processors {
		switch p := processor.(type) {
		case *localblocks.Processor:
			processors = append(processors, p)
		}
	}

	if i.queuebasedLocalBlocks != nil {
		processors = append(processors, i.queuebasedLocalBlocks)
	}

	i.processorsMtx.RUnlock()

	if len(processors) == 0 {
		return resp, fmt.Errorf("localblocks processor not found")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	unsafe := i.overrides.UnsafeQueryHints(i.instanceID)

	timeOverlapCutoff := i.cfg.Processor.LocalBlocks.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}

	e := traceql.NewEngine()

	// Compile the raw version of the query for head and wal blocks
	// These aren't cached and we put them all into the same evaluator
	// for efficiency.
	rawEval, err := e.CompileMetricsQueryRange(req, int(req.Exemplars), timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}

	// This is a summation version of the query for complete blocks
	// which can be cached. They are timeseries, so they need the job-level evaluator.
	jobEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
	if err != nil {
		return nil, err
	}

	for _, p := range processors {
		err = p.QueryRange(ctx, req, rawEval, jobEval)
		if err != nil {
			return nil, err
		}
	}

	// Combine the raw results into the job results
	walResults := rawEval.Results().ToProto(req)
	jobEval.ObserveSeries(walResults)

	r := jobEval.Results()
	rr := r.ToProto(req)
	return &tempopb.QueryRangeResponse{
		Series: rr,
	}, nil
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
