package querier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log/level"

	"github.com/grafana/tempo/pkg/cache"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

func (q *Querier) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	if req.QueryMode == QueryModeRecent {
		return q.queryRangeRecent(ctx, req)
	}

	return q.queryBlock(ctx, req)
}

func (q *Querier) queryRangeRecent(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	// correct max series limit logic should've been set by the query-frontend sharder
	c, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeSum, int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	results, err := q.forLiveStoreMetricsRing(ctx, func(ctx context.Context, client tempopb.MetricsClient) (any, error) {
		return client.QueryRange(ctx, req)
	})
	if err != nil {
		_ = level.Error(log.Logger).Log("msg", "error querying live-stores in Querier.queryRangeRecent", "err", err)
		return nil, fmt.Errorf("error querying live-stores in Querier.queryRangeRecent: %w", err)
	}

	for _, result := range results {
		resp := result.(*tempopb.QueryRangeResponse)
		c.Combine(resp)
		if c.MaxSeriesReached() {
			break
		}
	}

	return c.Response(), nil
}

// queryBlockSetup holds inputs derived once per request and shared by the
// cached and uncached evaluation paths.
type queryBlockSetup struct {
	tenantID          string
	meta              *backend.BlockMeta
	opts              common.SearchOptions
	expr              *traceql.RootExpr
	compileOpts       []traceql.CompileOption
	timeOverlapCutoff float64
	spanOnlyFetch     bool
	allowUnsafeHints  bool
}

func (q *Querier) prepareQueryBlock(ctx context.Context, req *tempopb.QueryRangeRequest) (*queryBlockSetup, error) {
	tenantID, err := validation.ExtractValidTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.queryBlock: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return nil, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:          req.Version,
		TenantID:         tenantID,
		StartTime:        time.Unix(0, int64(req.Start)),
		EndTime:          time.Unix(0, int64(req.End)),
		BlockID:          blockID,
		Size_:            req.Size_,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)

	// Parse without optimizations to read hints; optimizations are applied by CompileMetricsQueryRange.
	expr, err := traceql.ParseNoOptimizations(req.Query)
	if err != nil {
		return nil, err
	}

	var compileOpts []traceql.CompileOption
	unsafe := q.limits.UnsafeQueryHints(tenantID)
	if unsafe {
		compileOpts = append(compileOpts, traceql.WithUnsafeHints(true))
	}
	for _, name := range req.SkipASTTransformations {
		compileOpts = append(compileOpts, traceql.WithSkipOptimization(name))
	}

	timeOverlapCutoff := q.cfg.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}
	compileOpts = append(compileOpts, traceql.WithTimeOverlapCutoff(timeOverlapCutoff))

	var spanOnlyFetch bool
	if p := q.limits.MetricsSpanOnlyFetch(tenantID); p != nil {
		spanOnlyFetch = *p
		compileOpts = append(compileOpts, traceql.WithSpanOnlyFetch(spanOnlyFetch))
	}

	return &queryBlockSetup{
		tenantID:          tenantID,
		meta:              meta,
		opts:              opts,
		expr:              expr,
		compileOpts:       compileOpts,
		timeOverlapCutoff: timeOverlapCutoff,
		spanOnlyFetch:     spanOnlyFetch,
		allowUnsafeHints:  unsafe,
	}, nil
}

func (q *Querier) queryBlock(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	s, err := q.prepareQueryBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	queryHash := rowGroupQueryRangeHash(req, s.meta.DedicatedColumns, s.timeOverlapCutoff, s.spanOnlyFetch)
	elig := evaluateRowGroupCacheEligibility(req, s.expr, q.cacheProvider, s.allowUnsafeHints, queryHash)
	if !elig.use {
		// Only record a skip when caching was configured but bypassed for this
		// specific request; otherwise the metric fills with "no provider"
		// entries from deployments that haven't configured the cache at all.
		if elig.reason != cacheSkipReasonNoProvider {
			metricRowGroupCacheSkips.WithLabelValues(s.tenantID, elig.reason).Inc()
		}
		return q.queryBlockUncached(ctx, req, s)
	}
	return q.queryBlockWithRowGroupCache(ctx, req, s, queryHash, elig.cache)
}

// queryBlockUncached is the original single-evaluator path. Used when the
// per-row-group cache is unavailable or ineligible for this query.
func (q *Querier) queryBlockUncached(ctx context.Context, req *tempopb.QueryRangeRequest, s *queryBlockSetup) (*tempopb.QueryRangeResponse, error) {
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, s.compileOpts...)
	if err != nil {
		return nil, err
	}

	f := traceql.NewSpansetFetcherWrapperBoth(
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return q.store.Fetch(ctx, s.meta, req, s.opts)
		},
		func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return q.store.FetchSpans(ctx, s.meta, req, s.opts)
		},
	)

	err = eval.Do(ctx, f, uint64(s.meta.StartTime.UnixNano()), uint64(s.meta.EndTime.UnixNano()), int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	res := eval.Results()
	inspectedBytes, spansTotal, _ := eval.Metrics()
	return buildQueryRangeResponse(res, req, inspectedBytes, spansTotal), nil
}

// queryBlockWithRowGroupCache evaluates the block by caching the first-stage
// (Raw) result per row group. Hits and freshly-computed misses are merged via
// the Sum-mode QueryRangeCombiner — the same pattern the frontend uses across
// blocks.
func (q *Querier) queryBlockWithRowGroupCache(ctx context.Context, req *tempopb.QueryRangeRequest, s *queryBlockSetup, queryHash uint64, c cache.Cache) (*tempopb.QueryRangeResponse, error) {
	startPage := int(req.StartPage)
	numPages := int(req.PagesToSearch)
	if numPages <= 0 {
		return q.queryBlockUncached(ctx, req, s)
	}

	keys := make([]string, 0, numPages)
	keyToRG := make(map[string]int, numPages)
	rgIdxs := make([]int, 0, numPages)
	for i := 0; i < numPages; i++ {
		rgIdx := startPage + i
		key := metricsRowGroupCacheKey(s.tenantID, s.meta.BlockID, rgIdx, queryHash)
		if key == "" {
			return q.queryBlockUncached(ctx, req, s)
		}
		keys = append(keys, key)
		keyToRG[key] = rgIdx
		rgIdxs = append(rgIdxs, rgIdx)
	}

	foundKeys, foundBufs, _ := c.Fetch(ctx, keys)
	hits := make(map[int]*tempopb.QueryRangeResponse, len(foundKeys))
	for i, k := range foundKeys {
		resp, err := unmarshalRowGroupCacheValue(foundBufs[i])
		if err != nil {
			continue
		}
		if idx, ok := keyToRG[k]; ok {
			hits[idx] = resp
		}
	}

	missIdxs := make([]int, 0, numPages-len(hits))
	for _, rgIdx := range rgIdxs {
		if _, ok := hits[rgIdx]; !ok {
			missIdxs = append(missIdxs, rgIdx)
		}
	}

	metricRowGroupCacheHits.WithLabelValues(s.tenantID).Add(float64(len(hits)))
	metricRowGroupCacheMisses.WithLabelValues(s.tenantID).Add(float64(len(missIdxs)))

	// Merger runs the Sum stage, matching what the frontend QueryRangeCombiner
	// does across blocks. maxSeries=0 — truncation is applied after merge so
	// cache values stay independent of MaxSeries.
	combiner, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeSum, 0)
	if err != nil {
		return nil, err
	}

	// Feed cached hits first. Combine() also aggregates the InspectedBytes /
	// InspectedSpans counters carried inside each response's Metrics.
	for _, rgIdx := range rgIdxs {
		if resp, ok := hits[rgIdx]; ok {
			combiner.Combine(resp)
		}
	}

	if len(missIdxs) > 0 {
		responses, err := q.store.FetchSpansForRowGroups(ctx, s.meta, traceql.FetchSpansRequest{}, s.opts, missIdxs)
		if errors.Is(err, tempodb.ErrPerRowGroupUnsupported) {
			// Defensive — eligibility check should have caught this.
			metricRowGroupCacheSkips.WithLabelValues(s.tenantID, cacheSkipReasonWrongVersion).Inc()
			return q.queryBlockUncached(ctx, req, s)
		}
		if err != nil {
			return nil, fmt.Errorf("per-row-group fetch: %w", err)
		}
		if len(responses) != len(missIdxs) {
			return nil, fmt.Errorf("per-row-group fetch returned %d responses for %d requested row groups", len(responses), len(missIdxs))
		}

		// Sequential consumption is required for the FetchSpansForRowGroups
		// Bytes() callback to attribute bytes to the right row group.
		for i, rgIdx := range missIdxs {
			resp, err := q.evaluateRowGroup(ctx, req, s, responses[i])
			if err != nil {
				return nil, fmt.Errorf("evaluating row group %d: %w", rgIdx, err)
			}

			combiner.Combine(resp)

			key := metricsRowGroupCacheKey(s.tenantID, s.meta.BlockID, rgIdx, queryHash)
			if buf, mErr := marshalRowGroupCacheValue(resp); mErr == nil {
				rowGroupCacheStore(ctx, c, key, buf)
			}
		}
	}

	return truncateToMaxSeries(combiner.Response(), req), nil
}

// evaluateRowGroup builds a fresh Raw-mode evaluator over the given per-row-
// group fetch response. maxSeries is 0 — truncation runs after the Sum-stage
// merge so cache values are MaxSeries-agnostic.
func (q *Querier) evaluateRowGroup(ctx context.Context, req *tempopb.QueryRangeRequest, s *queryBlockSetup, fetchResp traceql.FetchSpansOnlyResponse) (*tempopb.QueryRangeResponse, error) {
	eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, s.compileOpts...)
	if err != nil {
		return nil, err
	}

	// We only have the span-only iterator from FetchSpansForRowGroups. The
	// spanset fallback path inside Do is never reached when the query is
	// span-level and DoSpansOnly succeeds; if it does need spansets we'd need
	// a richer per-row-group API. For now, surface that as an error so we
	// can detect it in tests.
	f := traceql.NewSpansetFetcherWrapperBoth(
		func(_ context.Context, _ traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
			return traceql.FetchSpansResponse{}, fmt.Errorf("spanset fetch not supported in per-row-group path")
		},
		func(_ context.Context, _ traceql.FetchSpansRequest) (traceql.FetchSpansOnlyResponse, error) {
			return fetchResp, nil
		},
	)

	err = eval.Do(ctx, f, uint64(s.meta.StartTime.UnixNano()), uint64(s.meta.EndTime.UnixNano()), 0)
	if err != nil {
		return nil, err
	}

	res := eval.Results()
	inspectedBytes, spansTotal, _ := eval.Metrics()
	return &tempopb.QueryRangeResponse{
		Series: res.ToProto(req),
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: inspectedBytes,
			InspectedSpans: spansTotal,
		},
	}, nil
}

func buildQueryRangeResponse(res traceql.SeriesSet, req *tempopb.QueryRangeRequest, inspectedBytes, spansTotal uint64) *tempopb.QueryRangeResponse {
	limited := res
	partial := false
	if req.MaxSeries > 0 && len(res) > int(req.MaxSeries) {
		limited = make(traceql.SeriesSet, int(req.MaxSeries))
		count := 0
		for k, v := range res {
			if count >= int(req.MaxSeries) {
				break
			}
			limited[k] = v
			count++
		}
		partial = true
	}
	response := &tempopb.QueryRangeResponse{
		Series: limited.ToProto(req),
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: inspectedBytes,
			InspectedSpans: spansTotal,
		},
	}
	if partial {
		response.Status = tempopb.PartialStatus_PARTIAL
	}
	return response
}

// truncateToMaxSeries trims the merged response to MaxSeries entries and marks
// it PARTIAL. Truncation runs after Sum-stage merge so cache values stay
// MaxSeries-agnostic.
func truncateToMaxSeries(resp *tempopb.QueryRangeResponse, req *tempopb.QueryRangeRequest) *tempopb.QueryRangeResponse {
	if req.MaxSeries <= 0 || resp == nil {
		return resp
	}
	if uint32(len(resp.Series)) <= req.MaxSeries {
		return resp
	}
	resp.Series = resp.Series[:req.MaxSeries]
	resp.Status = tempopb.PartialStatus_PARTIAL
	return resp
}
