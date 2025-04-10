package localblocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/modules/ingester"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

// QueryRange returns metrics.
func (p *Processor) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest, rawEval *traceql.MetricsEvaluator, jobEval *traceql.MetricsFrontendEvaluator) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p.blocksMtx.RLock()
	defer p.blocksMtx.RUnlock()

	cutoff := time.Now().Add(-p.Cfg.CompleteBlockTimeout).Add(-timeBuffer)
	if req.Start < uint64(cutoff.UnixNano()) {
		return fmt.Errorf("time range must be within last %v", p.Cfg.CompleteBlockTimeout)
	}

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return fmt.Errorf("compiling query: %w", err)
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

	withinRange := func(m *backend.BlockMeta) bool {
		start := uint64(m.StartTime.UnixNano())
		end := uint64(m.EndTime.UnixNano())
		return req.Start <= end && req.End >= start
	}

	var (
		wg     = boundedwaitgroup.New(concurrency)
		jobErr = atomic.Error{}
	)

	if p.headBlock != nil && withinRange(p.headBlock.BlockMeta()) {
		wg.Add(1)
		go func(w common.WALBlock) {
			defer wg.Done()
			err := p.queryRangeWALBlock(ctx, w, rawEval)
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
			err := p.queryRangeWALBlock(ctx, w, rawEval)
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

			jobEval.ObserveSeries(resp)
		}(b)
	}

	wg.Wait()

	if err := jobErr.Load(); err != nil {
		return err
	}

	return nil
}

func (p *Processor) queryRangeWALBlock(ctx context.Context, b common.WALBlock, eval *traceql.MetricsEvaluator) error {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "Processor.QueryRange.WALBlock", trace.WithAttributes(
		attribute.String("block", m.BlockID.String()),
		attribute.Int64("blockSize", int64(m.Size_)),
	))
	defer span.End()

	fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	return eval.Do(ctx, fetcher, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()))
}

func (p *Processor) queryRangeCompleteBlock(ctx context.Context, b *ingester.LocalBlock, req tempopb.QueryRangeRequest, timeOverlapCutoff float64, unsafe bool, exemplars int) ([]*tempopb.TimeSeries, error) {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "Processor.QueryRange.CompleteBlock", trace.WithAttributes(
		attribute.String("block", m.BlockID.String()),
		attribute.Int64("blockSize", int64(m.Size_)),
	))
	defer span.End()

	// Trim and align the request for this block. I.e. if the request is "Last Hour" we don't want to
	// cache the response for that, we want only the few minutes time range for this block. This has
	// size savings but the main thing is that the response is reuseable for any overlapping query.
	req.Start, req.End, req.Step = traceql.TrimToBlockOverlap(req.Start, req.End, req.Step, m.StartTime, m.EndTime)

	if req.Start >= req.End {
		// After alignment there is no overlap or something else isn't right
		return nil, nil
	}

	cached, name, err := p.queryRangeCacheGet(ctx, m, req)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cached", cached != nil))

	if cached != nil {
		return cached.Series, nil
	}

	// Not in cache or not cacheable, so execute
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(&req, exemplars, timeOverlapCutoff, unsafe)
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

	data, err := p.walR.Read(ctx, name, (uuid.UUID)(m.BlockID), m.TenantID, nil)
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

	return p.walW.Write(ctx, name, (uuid.UUID)(m.BlockID), m.TenantID, data, nil)
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
