package localblocks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/fasthash/fnv1a"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/v2/modules/ingester"
	"github.com/grafana/tempo/v2/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/v2/pkg/tempopb"
	"github.com/grafana/tempo/v2/pkg/traceql"
	"github.com/grafana/tempo/v2/tempodb/backend"
	"github.com/grafana/tempo/v2/tempodb/encoding/common"
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
	eval, err := e.CompileMetricsQueryRange(req, false, int(req.Exemplars), timeOverlapCutoff, unsafe)
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
			resp, err := p.queryRangeCompleteBlock(ctx, b, *req, timeOverlapCutoff, unsafe, int(req.Exemplars))
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

func (p *Processor) queryRangeCompleteBlock(ctx context.Context, b *ingester.LocalBlock, req tempopb.QueryRangeRequest, timeOverlapCutoff float64, unsafe bool, exemplars int) ([]*tempopb.TimeSeries, error) {
	m := b.BlockMeta()
	span, ctx := opentracing.StartSpanFromContext(ctx, "Processor.QueryRange.CompleteBlock", opentracing.Tags{
		"block":     m.BlockID,
		"blockSize": m.Size,
	})
	defer span.Finish()

	// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
	// cache the response for that, we want only the few minutes time range for this block. This has
	// size savings but the main thing is that the response is reuseable for any overlapping query.
	req.Start, req.End, req.Step = traceql.TrimToOverlap(req.Start, req.End, req.Step, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))

	if req.Start >= req.End {
		// After alignment there is no overlap or something else isn't right
		return nil, nil
	}

	cached, name, err := p.queryRangeCacheGet(ctx, m, req)
	if err != nil {
		return nil, err
	}

	span.SetTag("cached", cached != nil)

	if cached != nil {
		return cached.Series, nil
	}

	// Not in cache or not cacheable, so execute
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(&req, false, exemplars, timeOverlapCutoff, unsafe)
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
		if err != nil {
			return nil, fmt.Errorf("writing local query cache: %w", err)
		}
	}

	return results, nil
}

func (p *Processor) queryRangeCacheGet(ctx context.Context, m *backend.BlockMeta, req tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, string, error) {
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
	h := fnv1a.HashString64(req.Query)
	h = fnv1a.AddUint64(h, req.Start)
	h = fnv1a.AddUint64(h, req.End)
	h = fnv1a.AddUint64(h, req.Step)

	// TODO - caching for WAL blocks
	// Including trace count means we can safely cache results
	// for wal blocks which might receive new data
	// h = fnv1a.AddUint64(h, m.TotalObjects)

	return h
}
