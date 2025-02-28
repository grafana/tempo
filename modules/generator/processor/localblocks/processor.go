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
	"github.com/grafana/tempo/pkg/tracesizes"
	"github.com/grafana/tempo/tempodb"
	"go.opentelemetry.io/otel"

	gen "github.com/grafana/tempo/modules/generator/processor"
	"github.com/grafana/tempo/pkg/livetraces"
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

var tracer = otel.Tracer("modules/generator/processor/localblocks")

const (
	timeBuffer       = 5 * time.Minute
	maxFlushAttempts = 100
)

// ProcessorOverrides is just the set of overrides needed here.
type ProcessorOverrides interface {
	DedicatedColumns(string) backend.DedicatedColumns
	MaxLocalTracesPerUser(userID string) int
	MaxBytesPerTrace(string) int
	UnsafeQueryHints(string) bool
}

type Processor struct {
	tenant    string
	logger    kitlog.Logger
	Cfg       Config
	wal       *wal.WAL
	walR      backend.Reader
	walW      backend.Writer
	ctx       context.Context
	cancel    func()
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
	liveTraces    *livetraces.LiveTraces[*v1.ResourceSpans]
	traceSizes    *tracesizes.Tracker

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

	ctx, cancel := context.WithCancel(context.Background())

	p = &Processor{
		tenant:         tenant,
		logger:         log.WithUserID(tenant, log.Logger),
		Cfg:            cfg,
		wal:            wal,
		walR:           backend.NewReader(wal.LocalBackend()),
		walW:           backend.NewWriter(wal.LocalBackend()),
		overrides:      overrides,
		enc:            enc,
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]*ingester.LocalBlock{},
		flushqueue:     flushqueues.NewPriorityQueue(metricFlushQueueSize.WithLabelValues(tenant)),
		liveTraces:     livetraces.New[*v1.ResourceSpans](func(rs *v1.ResourceSpans) uint64 { return uint64(rs.Size()) }),
		traceSizes:     tracesizes.New(),
		ctx:            ctx,
		cancel:         cancel,
		wg:             sync.WaitGroup{},
		cache:          lru.New(100),
		writer:         writer,
	}

	startCompleteQueue(p.Cfg.Concurrency)
	defer func() {
		if err != nil {
			// In case of failing startup stop the queue
			// because Shutdown will not be called.
			stopCompleteQueue()
		}
	}()

	err = p.reloadBlocks()
	if err != nil {
		return nil, fmt.Errorf("replaying blocks: %w", err)
	}

	p.wg.Add(3)
	go p.cutLoop()
	go p.deleteLoop()
	go p.metricLoop()

	if p.writer != nil && p.Cfg.FlushToStorage {
		p.wg.Add(1)
		go p.flushLoop()
	}

	return p, nil
}

func (*Processor) Name() string {
	return Name
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	p.push(time.Now(), req)
}

func (p *Processor) DeterministicPush(ts time.Time, req *tempopb.PushSpansRequest) {
	if time.Since(ts) > p.Cfg.CompleteBlockTimeout {
		// Ignore data that is beyond retention
		return
	}

	// Wait for room in pipeline if needed
	for p.backpressure() {
	}

	p.push(ts, req)
}

func (p *Processor) backpressure() bool {
	if p.Cfg.MaxLiveTracesBytes > 0 {
		// Check live traces
		p.liveTracesMtx.Lock()
		liveTracesSize := p.liveTraces.Size()
		p.liveTracesMtx.Unlock()

		if liveTracesSize >= p.Cfg.MaxLiveTracesBytes {
			// Live traces exceeds the expected amount of data in
			// per wal flush, so wait a bit.
			select {
			case <-p.ctx.Done():
			case <-time.After(1 * time.Second):
			}

			metricBackPressure.WithLabelValues(reasonWaitingForLiveTraces).Inc()
			return true
		}
	}

	// Check outstanding wal blocks
	p.blocksMtx.RLock()
	count := len(p.walBlocks)
	p.blocksMtx.RUnlock()

	if count > 1 {
		// There are multiple outstanding WAL blocks that need completion
		// so wait a bit.
		select {
		case <-p.ctx.Done():
		case <-time.After(1 * time.Second):
		}

		metricBackPressure.WithLabelValues(reasonWaitingForWAL).Inc()
		return true
	}

	return false
}

func (p *Processor) push(ts time.Time, req *tempopb.PushSpansRequest) {
	p.liveTracesMtx.Lock()
	defer p.liveTracesMtx.Unlock()

	var (
		before  = p.liveTraces.Len()
		maxLen  = p.maxLiveTraces()
		maxSz   = p.overrides.MaxBytesPerTrace(p.tenant)
		batches = req.Batches
	)

	if p.Cfg.FilterServerSpans {
		batches = filterBatches(batches)
	}

	for _, batch := range batches {

		// Spans in the batch are for the same trace.
		// We use the first one.
		if len(batch.ScopeSpans) == 0 || len(batch.ScopeSpans[0].Spans) == 0 {
			continue
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
		// Doesn't assert trace size because that is done above using traceSizes
		// which tracks it across flushes.
		if !p.liveTraces.PushWithTimestampAndLimits(ts, traceID, batch, maxLen, 0) {
			metricDroppedTraces.WithLabelValues(p.tenant, reasonLiveTracesExceeded).Inc()
			continue
		}
	}

	after := p.liveTraces.Len()

	// Number of new traces is the delta
	metricTotalTraces.WithLabelValues(p.tenant).Add(float64(after - before))
}

// maxLiveTraces for the tenant, if enabled, and read from config in order of precedence.
func (p *Processor) maxLiveTraces() uint64 {
	if !p.Cfg.AssertMaxLiveTraces {
		return 0
	}

	if m := p.overrides.MaxLocalTracesPerUser(p.tenant); m > 0 {
		return uint64(m)
	}

	return p.Cfg.MaxLiveTraces
}

func (p *Processor) Shutdown(context.Context) {
	p.cancel()
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

	stopCompleteQueue()
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

		case <-p.ctx.Done():
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

		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Processor) flushLoop() {
	defer p.wg.Done()

	go func() {
		<-p.ctx.Done()
		p.flushqueue.Close()
	}()

	for {
		o := p.flushqueue.Dequeue()
		if o == nil {
			return
		}

		op := o.(*flushOp)
		op.attempts++

		if op.attempts > maxFlushAttempts {
			_ = level.Error(p.logger).Log("msg", "failed to flush block after max attempts", "tenant", p.tenant, "block", op.blockID, "attempts", op.attempts)

			// attempt to delete the block
			p.blocksMtx.Lock()
			err := p.wal.LocalBackend().ClearBlock(op.blockID, p.tenant)
			if err != nil {
				_ = level.Error(p.logger).Log("msg", "failed to clear corrupt block", "tenant", p.tenant, "block", op.blockID, "err", err)
			}
			delete(p.completeBlocks, op.blockID)
			p.blocksMtx.Unlock()

			continue
		}

		err := p.flushBlock(op.blockID)
		if err != nil {
			_ = level.Info(p.logger).Log("msg", "re-queueing block for flushing", "block", op.blockID, "attempts", op.attempts, "err", err)
			metricFailedFlushes.Inc()

			delay := op.backoff()
			op.at = time.Now().Add(delay)

			go func() {
				time.Sleep(delay)
				if _, err := p.flushqueue.Enqueue(op); err != nil {
					_ = level.Error(p.logger).Log("msg", "failed to requeue block for flushing", "err", err)
				}
			}()
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

		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Processor) completeBlock(id uuid.UUID) error {
	p.blocksMtx.RLock()
	b := p.walBlocks[id]
	p.blocksMtx.RUnlock()

	if b == nil {
		_ = level.Warn(p.logger).Log("msg", "local blocks processor WAL block disappeared before being completed", "id", id)
		return nil
	}

	// Now create a new block
	var (
		ctx    = p.ctx
		reader = backend.NewReader(p.wal.LocalBackend())
		writer = backend.NewWriter(p.wal.LocalBackend())
		cfg    = p.Cfg.Block
	)

	ctx, span := tracer.Start(ctx, "Processor.CompleteBlock")
	defer span.End()

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

	// Verify the WAL block still exists, it's possible that it went out of retention
	// while it was being completed.
	if _, ok := p.walBlocks[id]; !ok {
		// WAL block is gone.
		_ = level.Warn(p.logger).Log("msg", "local blocks processor WAL block disappeared while being completed, deleting complete block", "id", id)
		err := p.wal.LocalBackend().ClearBlock(id, p.tenant)
		if err != nil {
			_ = level.Error(p.logger).Log("msg", "failed to clear complete block after WAL disappeared", "tenant", p.tenant, "block", id, "err", err)
		}
		return nil
	}

	p.completeBlocks[id] = ingester.NewLocalBlock(ctx, newBlock, p.wal.LocalBackend())
	metricCompletedBlocks.WithLabelValues(p.tenant).Inc()

	// Queue for flushing
	if p.Cfg.FlushToStorage {
		if _, err := p.flushqueue.Enqueue(newFlushOp((uuid.UUID)(newMeta.BlockID))); err != nil {
			_ = level.Error(p.logger).Log("msg", "local blocks processor failed to enqueue block for flushing", "err", err)
		}
	}

	err = b.Clear()
	if err != nil {
		return err
	}
	delete(p.walBlocks, (uuid.UUID)(b.BlockMeta().BlockID))

	return nil
}

func (p *Processor) flushBlock(id uuid.UUID) error {
	p.blocksMtx.RLock()
	completeBlock := p.completeBlocks[id]
	p.blocksMtx.RUnlock()

	if completeBlock == nil {
		return nil
	}

	err := p.writer.WriteBlock(p.ctx, completeBlock)
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
	for keys, sh := range m.Series {
		h := []*tempopb.RawHistogram{}

		for bucket, count := range sh.Histogram.Buckets() {
			if count != 0 {
				rawHistorgram = &tempopb.RawHistogram{
					Bucket: uint64(bucket),
					Count:  uint64(count),
				}

				h = append(h, rawHistorgram)
			}
		}

		errCount = 0
		if errs, ok := m.Errors[keys]; ok {
			errCount = errs
		}

		resp.Metrics = append(resp.Metrics, &tempopb.SpanMetrics{
			LatencyHistogram: h,
			Series:           metricSeriesToProto(sh.Series),
			Errors:           uint64(errCount),
		})
	}

	return resp, nil
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

	cuttoff := time.Now().Add(-p.Cfg.CompleteBlockTimeout)

	for id, b := range p.walBlocks {
		if b.BlockMeta().EndTime.Before(cuttoff) {
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
		if !p.Cfg.FlushToStorage {
			if b.BlockMeta().EndTime.Before(cuttoff) {
				level.Info(p.logger).Log("msg", "deleting complete block", "block", id.String())
				err = p.wal.LocalBackend().ClearBlock(id, p.tenant)
				if err != nil {
					return err
				}
				delete(p.completeBlocks, id)
			}
			continue
		}

		flushedTime := b.FlushedTime()
		if flushedTime.IsZero() {
			continue
		}

		if b.BlockMeta().EndTime.Before(cuttoff) {
			level.Info(p.logger).Log("msg", "deleting flushed complete block", "block", id.String())
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
	metricLiveTraces.WithLabelValues(p.tenant).Set(float64(p.liveTraces.Len()))
	metricLiveTraceBytes.WithLabelValues(p.tenant).Set(float64(p.liveTraces.Size()))

	since := time.Now().Add(-p.Cfg.TraceIdlePeriod)
	tracesToCut := p.liveTraces.CutIdle(since, immediate)

	p.liveTracesMtx.Unlock()

	if len(tracesToCut) == 0 {
		return nil
	}

	// Sort by ID
	sort.Slice(tracesToCut, func(i, j int) bool {
		return bytes.Compare(tracesToCut[i].ID, tracesToCut[j].ID) == -1
	})

	for _, t := range tracesToCut {

		tr := &tempopb.Trace{
			ResourceSpans: t.Batches,
		}

		err := p.writeHeadBlock(t.ID, tr)
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

	// Get trace timestamp bounds
	var start, end uint64
	for _, b := range tr.ResourceSpans {
		for _, ss := range b.ScopeSpans {
			for _, s := range ss.Spans {
				if start == 0 || s.StartTimeUnixNano < start {
					start = s.StartTimeUnixNano
				}
				if s.EndTimeUnixNano > end {
					end = s.EndTimeUnixNano
				}
			}
		}
	}

	// Convert from unix nanos to unix seconds
	startSeconds := uint32(start / uint64(time.Second))
	endSeconds := uint32(end / uint64(time.Second))

	err := p.headBlock.AppendTrace(id, tr, startSeconds, endSeconds, p.Cfg.AdjustTimeRangeForSlack)
	if err != nil {
		return err
	}

	return nil
}

func (p *Processor) resetHeadBlock() error {
	meta := &backend.BlockMeta{
		BlockID:           backend.NewUUID(),
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

	id := (uuid.UUID)(p.headBlock.BlockMeta().BlockID)
	p.walBlocks[id] = p.headBlock
	metricCutBlocks.WithLabelValues(p.tenant).Inc()

	err = p.resetHeadBlock()
	if err != nil {
		return fmt.Errorf("failed to resetHeadBlock: %w", err)
	}

	level.Info(p.logger).Log("msg", "queueing wal block for completion", "block", id.String())
	if err := enqueueCompleteOp(p.ctx, p, id); err != nil {
		_ = level.Error(p.logger).Log("msg", "local blocks processor failed to enqueue block for completion", "err", err)
	}

	return nil
}

func (p *Processor) reloadBlocks() error {
	var (
		ctx = p.ctx
		t   = p.tenant
		l   = p.wal.LocalBackend()
		r   = backend.NewReader(l)
	)

	// ------------------------------------
	// wal blocks
	// ------------------------------------
	level.Info(p.logger).Log("msg", "reloading wal blocks")
	walBlocks, err := p.wal.RescanBlocks(0, log.Logger)
	if err != nil {
		return err
	}

	// Important - Must take the lock while adding to the block lists and enqueuing work,
	// but don't take it until after the slowest RescanBlocks step above to prevent blocking
	// other work for other tenants.  All of the below steps are fairly quick.
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	for _, blk := range walBlocks {
		meta := blk.BlockMeta()
		if meta.TenantID == p.tenant {
			level.Info(p.logger).Log("msg", "reloading wal block", "block", meta.BlockID.String())
			p.walBlocks[(uuid.UUID)(meta.BlockID)] = blk

			level.Info(p.logger).Log("msg", "queueing replayed wal block for completion", "block", meta.BlockID.String())
			if err := enqueueCompleteOp(p.ctx, p, uuid.UUID(meta.BlockID)); err != nil {
				_ = level.Error(p.logger).Log("msg", "local blocks processor failed to enqueue wal block for completion after replay", "err", err)
			}
		}
	}
	level.Info(p.logger).Log("msg", "reloaded wal blocks", "count", len(p.walBlocks))

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
		level.Info(p.logger).Log("msg", "no tenants found, skipping complete block replay")
		return nil
	}

	ids, _, err := r.Blocks(ctx, p.tenant)
	if err != nil {
		return err
	}
	level.Info(p.logger).Log("msg", "reloading complete blocks", "count", len(ids))

	for _, id := range ids {
		level.Info(p.logger).Log("msg", "reloading complete block", "block", id.String())
		meta, err := r.BlockMeta(ctx, id, t)

		var clearBlock bool
		if err != nil {
			var vv *json.SyntaxError
			if errors.Is(err, backend.ErrDoesNotExist) || errors.As(err, &vv) {
				clearBlock = true
			}
		}

		if clearBlock {
			level.Info(p.logger).Log("msg", "clearing block", "block", id.String(), "err", err)
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
		level.Info(p.logger).Log("msg", "reloaded complete block", "block", id.String())

		lb := ingester.NewLocalBlock(ctx, blk, l)
		p.completeBlocks[id] = lb

		if p.Cfg.FlushToStorage && lb.FlushedTime().IsZero() {
			level.Info(p.logger).Log("msg", "queueing reloaded block for flushing", "block", id.String())
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
		sum += b.BlockMeta().Size_
	}

	metricBlockSize.WithLabelValues(p.tenant).Set(float64(sum))
}

func metricSeriesToProto(series traceqlmetrics.MetricSeries) []*tempopb.KeyValue {
	var r []*tempopb.KeyValue
	for _, kv := range series {
		if kv.Key != "" {
			r = append(r, traceQLStaticToProto(&kv))
		}
	}
	return r
}

func traceQLStaticToProto(kv *traceqlmetrics.KeyValue) *tempopb.KeyValue {
	val := tempopb.TraceQLStatic{Type: int32(kv.Value.Type)}

	switch kv.Value.Type {
	case traceql.TypeInt:
		n, _ := kv.Value.Int()
		val.N = int64(n)
	case traceql.TypeFloat:
		val.F = kv.Value.Float()
	case traceql.TypeString:
		val.S = kv.Value.EncodeToString(false)
	case traceql.TypeBoolean:
		b, _ := kv.Value.Bool()
		val.B = b
	case traceql.TypeDuration:
		d, _ := kv.Value.Duration()
		val.D = uint64(d)
	case traceql.TypeStatus:
		st, _ := kv.Value.Status()
		val.Status = int32(st)
	case traceql.TypeKind:
		k, _ := kv.Value.Kind()
		val.Kind = int32(k)
	default:
		val = tempopb.TraceQLStatic{Type: int32(traceql.TypeNil)}
	}

	return &tempopb.KeyValue{Key: kv.Key, Value: &val}
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
