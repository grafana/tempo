package localblocks

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"
)

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

	e := traceql.NewEngine()

	// Compile the raw version of the query for wal blocks
	// These aren't cached and we put them all into the same evaluator
	// for efficiency.
	eval, err := e.CompileMetricsQueryRange(req, false, timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}

	// This is a summation version of the query for complete blocks
	// which can be cached. But we need their results separately so they are
	// computed separately.
	overallEvalMtx := sync.Mutex{}
	overallEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
	if err != nil {
		return nil, err
	}

	withinRange := func(m *backend.BlockMeta) bool {
		start := uint64(m.StartTime.UnixNano())
		end := uint64(m.EndTime.UnixNano())
		return req.Start < end && req.End > start
	}

	var (
		wg     = boundedwaitgroup.New(concurrency)
		jobErr = atomic.Error{}
	)

	if p.headBlock != nil && withinRange(p.headBlock.BlockMeta()) {
		wg.Add(1)
		go func(w common.WALBlock) {
			defer wg.Done()
			err := p.queryRangeWALBlock(ctx, w, eval)
			if err != nil {
				jobErr.Store(err)
			}
		}(p.headBlock)
	}

	for _, w := range p.walBlocks {
		if jobErr.Load() != nil {
			break
		}

		if !withinRange(w.BlockMeta()) {
			continue
		}

		wg.Add(1)
		go func(w common.WALBlock) {
			defer wg.Done()
			err := p.queryRangeWALBlock(ctx, w, eval)
			if err != nil {
				jobErr.Store(err)
			}
		}(w)
	}

	for _, b := range p.completeBlocks {
		if jobErr.Load() != nil {
			break
		}

		if !withinRange(b.BlockMeta()) {
			continue
		}

		wg.Add(1)
		go func(b *ingester.LocalBlock) {
			defer wg.Done()
			resp, err := p.queryRangeCompleteBlock(ctx, b, *req, timeOverlapCutoff, unsafe)
			if err != nil {
				jobErr.Store(err)
				return
			}

			overallEvalMtx.Lock()
			defer overallEvalMtx.Unlock()
			overallEval.ObserveSeries(resp)
		}(b)
	}

	wg.Wait()

	if err := jobErr.Load(); err != nil {
		return nil, err
	}

	// Combine the uncacheable results into the overall results
	walResults := eval.Results().ToProto(req)
	overallEval.ObserveSeries(walResults)

	return overallEval.Results(), nil
}

func (p *Processor) queryRangeWALBlock(ctx context.Context, b common.WALBlock, eval *traceql.MetricsEvalulator) error {
	m := b.BlockMeta()
	span, ctx := opentracing.StartSpanFromContext(ctx, "Processor.QueryRange.WALBlock", opentracing.Tags{
		"block":     m.BlockID,
		"blockSize": m.Size,
	})
	defer span.Finish()

	fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	return eval.Do(ctx, fetcher, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))
}

func (p *Processor) queryRangeCompleteBlock(ctx context.Context, b *ingester.LocalBlock, req tempopb.QueryRangeRequest, timeOverlapCutoff float64, unsafe bool) ([]*tempopb.TimeSeries, error) {
	m := b.BlockMeta()
	span, ctx := opentracing.StartSpanFromContext(ctx, "Processor.QueryRange.CompleteBlock", opentracing.Tags{
		"block":     m.BlockID,
		"blockSize": m.Size,
	})
	defer span.Finish()

	fmt.Println("Before:", m.BlockID, time.Unix(0, int64(req.Start)), time.Unix(0, int64(req.End)))
	fmt.Println("Block:", m.BlockID, m.StartTime, m.EndTime)

	// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
	// cache the response for that, we want only the few minutes time range for this block. This has
	// size savings but the main thing is that the response is reuseable for any overlapping query.
	// TODO - This buffer time is needed to close gaps between block meta time and actual
	// span contents
	blockStart := m.StartTime.Add(-time.Minute)
	blockEnd := m.EndTime.Add(time.Minute)

	req.Start = max(uint64(blockStart.UnixNano()), req.Start)
	req.End = min(uint64(blockEnd.UnixNano()), req.End)
	req.Start = (req.Start / req.Step) * req.Step
	req.End = (req.End/req.Step)*req.Step + req.Step

	fmt.Println("After:", m.BlockID, time.Unix(0, int64(req.Start)), time.Unix(0, int64(req.End)))

	cached, name, err := p.queryRangeCacheGet(ctx, m, req)
	if err != nil {
		return nil, err
	}

	span.SetTag("cached", cached != nil)

	if cached != nil {
		return cached.Series, nil
	}

	// Not in cache or not cacheable, so execute
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(&req, false, timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}
	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})
	err = eval.Do(ctx, f, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))
	if err != nil {
		return nil, err
	}

	results := eval.Results().ToProto(&req)

	if name != "" {
		err = p.queryRangeCacheSet(ctx, m, name, &tempopb.QueryRangeResponse{
			Series: results,
		})
	}

	return results, nil
}

func (p *Processor) queryRangeCacheGet(ctx context.Context, m *backend.BlockMeta, req tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, string, error) {
	cacheable := req.Start <= uint64(m.StartTime.UnixNano()) && req.End >= uint64(m.EndTime.UnixNano())
	if !cacheable {
		return nil, "", nil
	}

	hash := queryRangeHashForBlock(req)

	name := fmt.Sprintf("cache_query_range_%v.buf", hash)

	data, err := p.walR.Read(ctx, name, m.BlockID, m.TenantID, nil)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			// Not cached, but return the name/keypath so it can be set after
			return nil, name, nil
		}
		return nil, "", err
	}

	resp := &tempopb.QueryRangeResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return nil, "", err
	}

	return resp, name, nil
}

func (p *Processor) queryRangeCacheSet(ctx context.Context, m *backend.BlockMeta, name string, resp *tempopb.QueryRangeResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	return p.walW.Write(ctx, name, m.BlockID, m.TenantID, data, nil)
}

func queryRangeHashForBlock(req tempopb.QueryRangeRequest) uint64 {
	h := fnv.New64a()
	buf := make([]byte, 8)

	h.Write([]byte(req.Query))

	binary.BigEndian.PutUint64(buf, req.Step)
	h.Write(buf)

	binary.BigEndian.PutUint64(buf, req.Start)
	h.Write(buf)

	binary.BigEndian.PutUint64(buf, req.End)
	h.Write(buf)

	// TODO - caching for WAL blocks
	// Including trace count means we can safely cache results
	// for wal blocks which might receive new data
	// binary.BigEndian.PutUint64(buf, uint64(m.TotalObjects))
	// h.Write(buf)

	return h.Sum64()
}
