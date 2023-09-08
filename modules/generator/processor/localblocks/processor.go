package localblocks

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/golang/groupcache/lru"
	"github.com/google/uuid"
	gen "github.com/grafana/tempo/modules/generator/processor"
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
	"github.com/pkg/errors"
)

type Processor struct {
	tenant   string
	Cfg      Config
	wal      *wal.WAL
	closeCh  chan struct{}
	wg       sync.WaitGroup
	cacheMtx sync.RWMutex
	cache    *lru.Cache

	blocksMtx      sync.RWMutex
	headBlock      common.WALBlock
	walBlocks      map[uuid.UUID]common.WALBlock
	completeBlocks map[uuid.UUID]common.BackendBlock
	lastCutTime    time.Time

	liveTracesMtx sync.Mutex
	liveTraces    *liveTraces
}

var _ gen.Processor = (*Processor)(nil)

func New(cfg Config, tenant string, wal *wal.WAL) (*Processor, error) {
	if wal == nil {
		return nil, errors.New("local blocks processor requires traces wal")
	}

	p := &Processor{
		Cfg:            cfg,
		tenant:         tenant,
		wal:            wal,
		walBlocks:      map[uuid.UUID]common.WALBlock{},
		completeBlocks: map[uuid.UUID]common.BackendBlock{},
		liveTraces:     newLiveTraces(),
		closeCh:        make(chan struct{}),
		wg:             sync.WaitGroup{},
		cache:          lru.New(100),
	}

	err := p.reloadBlocks()
	if err != nil {
		return nil, errors.Wrap(err, "replaying blocks")
	}

	p.wg.Add(4)
	go p.flushLoop()
	go p.deleteLoop()
	go p.completeLoop()
	go p.metricLoop()

	return p, nil
}

func (*Processor) Name() string {
	return "LocalBlocksProcessor"
}

func (p *Processor) PushSpans(_ context.Context, req *tempopb.PushSpansRequest) {
	p.liveTracesMtx.Lock()
	defer p.liveTracesMtx.Unlock()

	before := p.liveTraces.Len()

	for _, batch := range req.Batches {
		if batch = filterBatch(batch); batch != nil {
			err := p.liveTraces.Push(batch, p.Cfg.MaxLiveTraces)
			if errors.Is(err, errMaxExceeded) {
				metricDroppedTraces.WithLabelValues(p.tenant, reasonLiveTracesExceeded).Inc()
			}
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
		level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut remaining traces on shutdown", "err", err)
	}

	err = p.cutBlocks(true)
	if err != nil {
		level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut head block on shutdown", "err", err)
	}
}

func (p *Processor) flushLoop() {
	defer p.wg.Done()

	flushTicker := time.NewTicker(p.Cfg.FlushCheckPeriod)
	defer flushTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			err := p.cutIdleTraces(false)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut idle traces", "err", err)
			}

			err = p.cutBlocks(false)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to cut head block", "err", err)
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
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to delete old blocks", "err", err)
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
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to complete a block", "err", err)
			}

		case <-p.closeCh:
			return
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
		enc    = encoding.DefaultEncoding()
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

	newMeta, err := enc.CreateBlock(ctx, cfg, b.BlockMeta(), iter, reader, writer)
	if err != nil {
		return err
	}

	newBlock, err := enc.OpenBlock(newMeta, reader)
	if err != nil {
		return err
	}

	// Add new block and delete old block
	p.blocksMtx.Lock()
	defer p.blocksMtx.Unlock()

	p.completeBlocks[newMeta.BlockID] = newBlock

	err = b.Clear()
	if err != nil {
		return err
	}
	delete(p.walBlocks, b.BlockMeta().BlockID)

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
			err = b.Clear()
			if err != nil {
				return err
			}
			delete(p.walBlocks, id)
		}
	}

	for id, b := range p.completeBlocks {
		if b.BlockMeta().EndTime.Before(before) {
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
	if immediate {
		since = time.Time{}
	}

	tracesToCut := p.liveTraces.CutIdle(since)

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
	block, err := p.wal.NewBlock(uuid.New(), p.tenant, model.CurrentEncoding)
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

	// Final flush
	err := p.headBlock.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush head block: %w", err)
	}

	p.walBlocks[p.headBlock.BlockMeta().BlockID] = p.headBlock

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

	ids, err := r.Blocks(ctx, p.tenant)
	if err != nil {
		return err
	}

	for _, id := range ids {
		meta, err := r.BlockMeta(ctx, id, t)

		if errors.Is(err, backend.ErrDoesNotExist) {
			// Partially written block, delete and continue
			err = l.ClearBlock(id, t)
			if err != nil {
				level.Error(log.WithUserID(p.tenant, log.Logger)).Log("msg", "local blocks processor failed to clear partially written block during replay", "err", err)
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

		p.completeBlocks[id] = blk
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

// filterBatch to only spans with kind==server. Does not modify the input
// but returns a new struct referencing the same input pointers. Returns nil
// if there were no matching spans.
func filterBatch(batch *v1.ResourceSpans) *v1.ResourceSpans {
	var keepSS []*v1.ScopeSpans
	for _, ss := range batch.ScopeSpans {

		var keepSpans []*v1.Span
		for _, s := range ss.Spans {
			if s.Kind == v1.Span_SPAN_KIND_SERVER {
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
		return &v1.ResourceSpans{
			Resource:   batch.Resource,
			ScopeSpans: keepSS,
		}
	}

	return nil
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
