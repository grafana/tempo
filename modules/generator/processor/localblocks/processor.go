package localblocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/groupcache/lru"
	"github.com/google/uuid"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/flushqueues"
	"github.com/grafana/tempo/tempodb"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/traceqlmetrics"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/grafana/tempo/tempodb/wal"
)

const timeBuffer = 5 * time.Minute

// ProcessorOverrides is just the set of overrides needed here.
type ProcessorOverrides interface {
	DedicatedColumns(string) backend.DedicatedColumns
	MaxBytesPerTrace(string) int
	UnsafeQueryHints(string) bool
}

type Processor struct {
	tenant    string
	logger    kitlog.Logger
	Cfg       Config
	wal       *wal.WAL
	closeCh   chan struct{}
	wg        sync.WaitGroup
	cacheMtx  sync.RWMutex
	cache     *lru.Cache
	overrides ProcessorOverrides
	enc       encoding.VersionedEncoding

	blocksMtx      sync.RWMutex
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]*ingester.LocalBlock
	lastCutTime    time.Time

	flushqueue *flushqueues.PriorityQueue

	liveTracesMtx sync.Mutex
	liveTraces    *liveTraces
	traceSizes    *traceSizes

	writer tempodb.Writer
}

var _ gen.Processor = (*Processor)(nil)

func New(cfg Config, tenant string, wal *wal.WAL, writer tempodb.Writer, overrides ProcessorOverrides) (p *Processor, err error) {
	if wal == nil {
		return nil, errors.New("local blocks processor requires traces wal")
	}

	enc := encoding.DefaultEncoding()
	if cfg.Block.Version != "" {
		enc, err = encoding.FromVersion(cfg.Block.Version)
		if err != nil {
			return nil, err
		}
	}

	p = &Processor{
		tenant:         tenant,
		logger:         log.WithUserID(tenant, log.Logger),
		Cfg:            cfg,
		wal:            wal,
		overrides:      overrides,
		enc:            enc,
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]*ingester.LocalBlock{},
		flushqueue:     flushqueues.NewPriorityQueue(metricFlushQueueSize.WithLabelValues(tenant)),
		liveTraces:     newLiveTraces(),
		traceSizes:     newTraceSizes(),
		closeCh:        make(chan struct{}),
		wg:             sync.WaitGroup{},
		cache:          lru.New(100),
		writer:         writer,
	}

	err = p.reloadBlocks()
	if err != nil {
		return nil, fmt.Errorf("replaying blocks: %w", err)
	}

	p.wg.Add(4)
	go p.cutLoop()
	go p.completeLoop()
	go p.deleteLoop()
	go p.metricLoop()

	if p.writer != nil && p.Cfg.FlushToStorage {
		p.wg.Add(1)
		go p.flushLoop()
	}

	return p, nil
}

func (*Processor) Name() string {
	return "LocalBlocksProcessor"
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	p.liveTracesMtx.Lock()
	defer p.liveTracesMtx.Unlock()

	before := p.liveTraces.Len()

	maxSz := p.overrides.MaxBytesPerTrace(p.tenant)

	batches := req.Batches
	if p.Cfg.FilterServerSpans {
		batches = filterBatches(batches)
	}

	for _, batch := range batches {

		// Spans in the batch are for the same trace.
		// We use the first one.
		if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
			return
		}
		traceID := batch.ScopeSpans[0].Spans[0].TraceId

		// Metric total spans regardless of outcome
		numSpans := 0
		for _, ss := range batch.ScopeSpans {
			numSpans += len(ss.Spans)
		}
		metricTotalSpans.WithLabelValues(p.tenant).Add(float64(numSpans))

		// Check max trace size
		if maxSz > 0 && !p.traceSizes.Allow(traceID, batch.Size(), maxSz) {
			metricDroppedSpans.WithLabelValues(p.tenant, reasonTraceSizeExceeded).Add(float64(numSpans))
			continue
		}

		// Live traces
		if !p.liveTraces.Push(traceID, batch, p.Cfg.MaxLiveTraces) {
			metricDroppedTraces.WithLabelValues(p.tenant, reasonLiveTracesExceeded).Inc()
			continue
		}

	}

	after := p.liveTraces.Len()

	// Number of new traces is the delta
	metricTotalTraces.WithLabelValues(p.tenant).Add(float64(after - before))
}

func (p *Processor) Shutdown(context.Context) {
	close(p.closeCh)
	p.wg.Wait()

	// Immediately cut all traces from memory
	err := p.cutIdleTraces(true)
	if err != nil {
		level.Error(p.logger).Log("msg", "local blocks processor failed to cut remaining traces on shutdown", "err", err)
	}

	err = p.cutBlocks(true)
	if err != nil {
		level.Error(p.logger).Log("msg", "local blocks processor failed to cut head block on shutdown", "err", err)
	}
}

func (p *Processor) cutLoop() {
	defer p.wg.Done()

	flushTicker := time.NewTicker(p.Cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			err := p.cutIdleTraces(false)
			if err != nil {
				level.Error(p.logger).Log("msg", "local blocks processor failed to cut idle traces", "err", err)
			}

			err = p.cutBlocks(false)
			if err != nil {
				level.Error(p.logger).Log("msg", "local blocks processor failed to cut head block", "err", err)
			}

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) deleteLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := p.deleteOldBlocks()
			if err != nil {
				level.Error(p.logger).Log("msg", "local blocks processor failed to delete old blocks", "err", err)
			}

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) completeLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := p.completeBlock()
			if err != nil {
				level.Error(p.logger).Log("msg", "local blocks processor failed to complete a block", "err", err)
			}

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) flushLoop() {
	defer p.wg.Done()

	go func() {
		<-p.closeCh
		p.flushqueue.Close()
	}()

	for {
		o := p.flushqueue.Dequeue()
		if o == nil {
			return
		}

		op := o.(*flushOp)
		op.attempts++
		err := p.flushBlock(op.blockID)
		if err != nil {
			_ = level.Error(p.logger).Log("msg", "failed to flush a block", "err", err)

			_ = level.Info(p.logger).Log("msg", "re-queueing block for flushing", "block", op.blockID, "attempts", op.attempts)
			op.at = time.Now().Add(op.backoff())

			if _, err := p.flushqueue.Enqueue(op); err != nil {
				_ = level.Error(p.logger).Log("msg", "failed to requeue block for flushing", "err", err)
			}
		}
	}
}

func (p *Processor) metricLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Instead of reacting to every block flush/update, just run on a timer.
			p.recordBlockBytes()

		case <-p.closeCh:
			return
		}
	}
}

func (p *Processor) completeBlock() error {
	// Get a wal block
	var firstWalBlock common.WALBlock
	p.blocksMtx.RLock()
	for _, e := range p.walBlocks {
		firstWalBlock = e
		break
	}
	p.blocksMtx.RUnlock()

	if firstWalBlock == nil {
		return nil
	}

	// Now create a new block
	var (
		ctx    = context.Background()
		reader = backend.NewReader(p.wal.LocalBackend())
		writer = backend.NewWriter(p.wal.LocalBackend())
		cfg    = p.Cfg.Block
		b      = firstWalBlock
	)

	iter, err := b.Iterator()
	if err != nil {
		return err
	}
	defer iter.Close()

	newMeta, err := p.enc.CreateBlock(ctx, cfg, b.BlockMeta(), iter, reader, writer)
	if err != nil {
		return err
	}

	newBlock, err := p.enc.OpenBlock(newMeta, reader)
	if err != nil {
		return err
	}

	// Add new block and delete old block
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	p.completeBlocks[newMeta.BlockID] = ingester.NewLocalBlock(ctx, newBlock, p.wal.LocalBackend())
	metricCompletedBlocks.WithLabelValues(p.tenant).Inc()

	// Queue for flushing
	if _, err := p.flushqueue.Enqueue(newFlushOp(newMeta.BlockID)); err != nil {
		_ = level.Error(p.logger).Log("msg", "local blocks processor failed to enqueue block for flushing", "err", err)
	}

	err = b.Clear()
	if err != nil {
		return err
	}
	delete(p.walBlocks, b.BlockMeta().BlockID)

	return nil
}

func (p *Processor) flushBlock(id uuid.UUID) error {
	p.blocksMtx.RLock()
	completeBlock := p.completeBlocks[id]
	p.blocksMtx.RUnlock()

	if completeBlock == nil {
		return nil
	}

	ctx := context.Background()
	err := p.writer.WriteBlock(ctx, completeBlock)
	if err != nil {
		return err
	}
	metricFlushedBlocks.WithLabelValues(p.tenant).Inc()
	_ = level.Info(p.logger).Log("msg", "flushed block to storage", "block", id.String())
	return nil
}

func (p *Processor) GetMetrics(ctx context.Context, req *tempopb.SpanMetricsRequest) (*tempopb.SpanMetricsResponse, error) {
	p.blocksMtx.RLock()
	defer p.blocksMtx.RUnlock()

	var (
		err       error
		startNano = uint64(time.Unix(int64(req.Start), 0).UnixNano())
		endNano   = uint64(time.Unix(int64(req.End), 0).UnixNano())
	)

	if startNano > 0 && endNano > 0 {
		cutoff := time.Now().Add(-p.Cfg.CompleteBlockTimeout).Add(-timeBuffer)
		if startNano < uint64(cutoff.UnixNano()) {
			return nil, fmt.Errorf("time range must be within last %v", p.Cfg.CompleteBlockTimeout)
		}
	}

	// Blocks to check
	blocks := make([]common.BackendBlock, 0, 1+len(p.walBlocks)+len(p.completeBlocks))
	if p.headBlock != nil {
		blocks = append(blocks, p.headBlock)
	}
	for _, b := range p.walBlocks {
		blocks = append(blocks, b)
	}
	for _, b := range p.completeBlocks {
		blocks = append(blocks, b)
	}

	m := traceqlmetrics.NewMetricsResults()
	for _, b := range blocks {

		var (
			meta       = b.BlockMeta()
			blockStart = uint32(meta.StartTime.Unix())
			blockEnd   = uint32(meta.EndTime.Unix())
			// We can only cache the results of this query on this block
			// if the time range fully covers this block (not partial).
			cacheable = req.Start <= blockStart && req.End >= blockEnd
			// Including the trace count in the cache key means we can safely
			// cache results for a wal block which can receive new data
			key = fmt.Sprintf("b:%s-c:%d-q:%s-g:%s", meta.BlockID.String(), meta.TotalObjects, req.Query, req.GroupBy)
		)

		var r *traceqlmetrics.MetricsResults

		if cacheable {
			r = p.metricsCacheGet(key)
		}

		// Uncacheable or not found in cache
		if r == nil {

			f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return b.Fetch(ctx, req, common.DefaultSearchOptions())
			})

			r, err = traceqlmetrics.GetMetrics(ctx, req.Query, req.GroupBy, 0, startNano, endNano, f)
			if err != nil {
				return nil, err
			}

			if cacheable {
				p.metricsCacheSet(key, r)
			}
		}

		m.Combine(r)

		if req.Limit > 0 && m.SpanCount >= int(req.Limit) {
			break
		}
	}

	resp := &tempopb.SpanMetricsResponse{
		SpanCount: uint64(m.SpanCount),
		Estimated: m.Estimated,
		Metrics:   make([]*tempopb.SpanMetrics, 0, len(m.Series)),
	}

	var rawHistorgram *tempopb.RawHistogram
	var errCount int
	for series, hist := range m.Series {
		h := []*tempopb.RawHistogram{}

		for bucket, count := range hist.Buckets() {
			if count != 0 {
				rawHistorgram = &tempopb.RawHistogram{
					Bucket: uint64(bucket),
					Count:  uint64(count),
				}

				h = append(h, rawHistorgram)
			}
		}

		errCount = 0
		if errs, ok := m.Errors[series]; ok {
			errCount = errs
		}

		resp.Metrics = append(resp.Metrics, &tempopb.SpanMetrics{
			LatencyHistogram: h,
			Series:           metricSeriesToProto(series),
			Errors:           uint64(errCount),
		})
	}

	return resp, nil
}

// QueryRange returns metrics.
func (p *Processor) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (traceql.SeriesSet, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p.blocksMtx.RLock()
	defer p.blocksMtx.RUnlock()

	cutoff := time.Now().Add(-p.Cfg.CompleteBlockTimeout).Add(-timeBuffer)
	if req.Start < uint64(cutoff.UnixNano()) {
		return nil, fmt.Errorf("time range must be within last %v", p.Cfg.CompleteBlockTimeout)
	}

	// Blocks to check
	blocks := make([]common.BackendBlock, 0, 1+len(p.walBlocks)+len(p.completeBlocks))
	if p.headBlock != nil {
		blocks = append(blocks, p.headBlock)
	}
	for _, b := range p.walBlocks {
		blocks = append(blocks, b)
	}
	for _, b := range p.completeBlocks {
		blocks = append(blocks, b)
	}
	if len(blocks) == 0 {
		return nil, nil
	}

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	unsafe := p.overrides.UnsafeQueryHints(p.tenant)

	timeOverlapCutoff := p.Cfg.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}

	concurrency := p.Cfg.Metrics.ConcurrentBlocks
	if v, ok := expr.Hints.GetInt(traceql.HintConcurrentBlocks, unsafe); ok && v > 0 && v < 100 {
		concurrency = uint(v)
	}

	// Compile the sharded version of the query
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, false, timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}

	var (
		wg     = boundedwaitgroup.New(concurrency)
		jobErr = atomic.Error{}
	)

	for _, b := range blocks {
		// If a job errored then quit immediately.
		if err := jobErr.Load(); err != nil {
			return nil, err
		}

		start := uint64(b.BlockMeta().StartTime.UnixNano())
		end := uint64(b.BlockMeta().EndTime.UnixNano())
		if start > req.End || end < req.Start {
			// Out of time range
			continue
		}

		wg.Add(1)
		go func(b common.BackendBlock) {
			defer wg.Done()

			m := b.BlockMeta()

			span, ctx := opentracing.StartSpanFromContext(ctx, "Processor.QueryRange.Block", opentracing.Tags{
				"block":     m.BlockID,
				"blockSize": m.Size,
			})
			defer span.Finish()

			// TODO - caching
			f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
				return b.Fetch(ctx, req, common.DefaultSearchOptions())
			})

			err := eval.Do(ctx, f, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))
			if err != nil {
				jobErr.Store(err)
			}
		}(b)
	}

	wg.Wait()

	if err := jobErr.Load(); err != nil {
		return nil, err
	}

	return eval.Results(), nil
}

func (p *Processor) metricsCacheGet(key string) *traceqlmetrics.MetricsResults {
	p.cacheMtx.RLock()
	defer p.cacheMtx.RUnlock()

	if r, ok := p.cache.Get(key); ok {
		return r.(*traceqlmetrics.MetricsResults)
	}

	return nil
}

func (p *Processor) metricsCacheSet(key string, m *traceqlmetrics.MetricsResults) {
	p.cacheMtx.Lock()
	defer p.cacheMtx.Unlock()

	p.cache.Add(key, m)
}

func (p *Processor) deleteOldBlocks() (err error) {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	before := time.Now().Add(-p.Cfg.CompleteBlockTimeout)

	for id, b := range p.walBlocks {
		if b.BlockMeta().EndTime.Before(before) {
			if _, ok := p.completeBlocks[id]; !ok {
				level.Warn(p.logger).Log("msg", "deleting WAL block that was never completed", "block", id.String())
			}
			err = b.Clear()
			if err != nil {
				return err
			}
			delete(p.walBlocks, id)
		}
	}

	for id, b := range p.completeBlocks {
		flushedTime := b.FlushedTime()
		if flushedTime.IsZero() {
			continue
		}

		if flushedTime.Add(p.Cfg.CompleteBlockTimeout).Before(time.Now()) {
			err = p.wal.LocalBackend().ClearBlock(id, p.tenant)
			if err != nil {
				return err
			}
			delete(p.completeBlocks, id)
		}
	}

	return
}

func (p *Processor) cutIdleTraces(immediate bool) error {
	p.liveTracesMtx.Lock()

	// Record live traces before flushing so we know the high water mark
	metricLiveTraces.WithLabelValues(p.tenant).Set(float64(len(p.liveTraces.traces)))

	since := time.Now().Add(-p.Cfg.TraceIdlePeriod)
	tracesToCut := p.liveTraces.CutIdle(since, immediate)

	p.liveTracesMtx.Unlock()

	if len(tracesToCut) == 0 {
		return nil
	}

	// Sort by ID
	sort.Slice(tracesToCut, func(i, j int) bool {
		return bytes.Compare(tracesToCut[i].id, tracesToCut[j].id) == -1
	})

	for _, t := range tracesToCut {

		tr := &tempopb.Trace{
			Batches: t.Batches,
		}

		err := p.writeHeadBlock(t.id, tr)
		if err != nil {
			return err
		}

	}

	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()
	return p.headBlock.Flush()
}

func (p *Processor) writeHeadBlock(id common.ID, tr *tempopb.Trace) error {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	if p.headBlock == nil {
		err := p.resetHeadBlock()
		if err != nil {
			return err
		}
	}

	now := uint32(time.Now().Unix())

	err := p.headBlock.AppendTrace(id, tr, now, now)
	if err != nil {
		return err
	}

	return nil
}

func (p *Processor) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           uuid.New(),
		TenantID:          p.tenant,
		DedicatedColumns:  p.overrides.DedicatedColumns(p.tenant),
		ReplicationFactor: backend.MetricsGeneratorReplicationFactor,
	}
	block, err := p.wal.NewBlock(meta, model.CurrentEncoding)
	if err != nil {
		return err
	}
	p.headBlock = block
	p.lastCutTime = time.Now()
	return nil
}

func (p *Processor) cutBlocks(immediate bool) error {
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	if p.headBlock == nil || p.headBlock.DataLength() == 0 {
		return nil
	}

	if !immediate && time.Since(p.lastCutTime) < p.Cfg.MaxBlockDuration && p.headBlock.DataLength() < p.Cfg.MaxBlockBytes {
		return nil
	}

	// Clear historical trace sizes for traces that weren't seen in this block.
	p.traceSizes.ClearIdle(p.lastCutTime)

	// Final flush
	err := p.headBlock.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush head block: %w", err)
	}

	p.walBlocks[p.headBlock.BlockMeta().BlockID] = p.headBlock
	metricCutBlocks.WithLabelValues(p.tenant).Inc()

	err = p.resetHeadBlock()
	if err != nil {
		return fmt.Errorf("failed to resetHeadBlock: %w", err)
	}

	return nil
}

func (p *Processor) reloadBlocks() error {
	var (
		ctx = context.Background()
		t   = p.tenant
		l   = p.wal.LocalBackend()
		r   = backend.NewReader(l)
	)

	// ------------------------------------
	// wal blocks
	// ------------------------------------
	walBlocks, err := p.wal.RescanBlocks(0, log.Logger)
	if err != nil {
		return err
	}
	for _, blk := range walBlocks {
		meta := blk.BlockMeta()
		if meta.TenantID == p.tenant {
			p.walBlocks[blk.BlockMeta().BlockID] = blk
		}
	}

	// ------------------------------------
	// Complete blocks
	// ------------------------------------

	// This is a quirk of the local backend, we shouldn't try to list
	// blocks until after we've made sure the tenant folder exists.
	tenants, err := r.Tenants(ctx)
	if err != nil {
		return err
	}
	if len(tenants) == 0 {
		return nil
	}

	ids, _, err := r.Blocks(ctx, p.tenant)
	if err != nil {
		return err
	}

	for _, id := range ids {
		meta, err := r.BlockMeta(ctx, id, t)

		var clearBlock bool
		if err != nil {
			var vv *json.SyntaxError
			if errors.Is(err, backend.ErrDoesNotExist) || errors.As(err, &vv) {
				clearBlock = true
			}
		}

		if clearBlock {
			// Partially written block, delete and continue
			err = l.ClearBlock(id, t)
			if err != nil {
				level.Error(p.logger).Log("msg", "local blocks processor failed to clear partially written block during replay", "err", err)
			}
			continue
		}

		if err != nil {
			return err
		}

		blk, err := encoding.OpenBlock(meta, r)
		if err != nil {
			return err
		}

		lb := ingester.NewLocalBlock(ctx, blk, l)
		p.completeBlocks[id] = lb

		if lb.FlushedTime().IsZero() {
			if _, err := p.flushqueue.Enqueue(newFlushOp(id)); err != nil {
				_ = level.Error(p.logger).Log("msg", "local blocks processor failed to enqueue block for flushing during replay", "err", err)
			}
		}
	}

	return nil
}

func (p *Processor) recordBlockBytes() {
	p.blocksMtx.RLock()
	defer p.blocksMtx.RUnlock()

	sum := uint64(0)

	if p.headBlock != nil {
		sum += p.headBlock.DataLength()
	}
	for _, b := range p.walBlocks {
		sum += b.DataLength()
	}
	for _, b := range p.completeBlocks {
		sum += b.BlockMeta().Size
	}

	metricBlockSize.WithLabelValues(p.tenant).Set(float64(sum))
}

func metricSeriesToProto(series traceqlmetrics.MetricSeries) []*tempopb.KeyValue {
	var r []*tempopb.KeyValue
	for _, kv := range series {
		if kv.Key != "" {
			static := kv.Value
			r = append(r, &tempopb.KeyValue{
				Key: kv.Key,
				Value: &tempopb.TraceQLStatic{
					Type:   int32(static.Type),
					N:      int64(static.N),
					F:      static.F,
					S:      static.S,
					B:      static.B,
					D:      uint64(static.D),
					Status: int32(static.Status),
					Kind:   int32(static.Kind),
				},
			})
		}
	}
	return r
}

// filterBatches to only root spans or kind==server. Does not modify the input
// but returns a new struct referencing the same input pointers. Returns nil
// if there were no matching spans.
func filterBatches(batches []*v1.ResourceSpans) []*v1.ResourceSpans {
	keep := make([]*v1.ResourceSpans, 0, len(batches))

	for _, batch := range batches {
		var keepSS []*v1.ScopeSpans
		for _, ss := range batch.ScopeSpans {

			var keepSpans []*v1.Span
			for _, s := range ss.Spans {
				if s.Kind == v1.Span_SPAN_KIND_SERVER || len(s.ParentSpanId) == 0 {
					keepSpans = append(keepSpans, s)
				}
			}

			if len(keepSpans) > 0 {
				keepSS = append(keepSS, &v1.ScopeSpans{
					Scope: ss.Scope,
					Spans: keepSpans,
				})
			}
		}

		if len(keepSS) > 0 {
			keep = append(keep, &v1.ResourceSpans{
				Resource:   batch.Resource,
				ScopeSpans: keepSS,
			})
		}
	}

	return keep
}

type flushOp struct {
	blockID  uuid.UUID
	at       time.Time // When to execute
	attempts int
	bo       time.Duration
}

func newFlushOp(blockID uuid.UUID) *flushOp {
	return &flushOp{
		blockID: blockID,
		at:      time.Now(),
		bo:      30 * time.Second,
	}
}

func (f *flushOp) Key() string { return f.blockID.String() }

func (f *flushOp) Priority() int64 { return -f.at.Unix() }

const maxBackoff = 5 * time.Minute

func (f *flushOp) backoff() time.Duration {
	f.bo *= 2
	if f.bo > maxBackoff {
		f.bo = maxBackoff
	}

	return f.bo
}
