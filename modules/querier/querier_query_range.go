package querier

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/pkg/util/log"
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
	// Get results from all generators
	replicationSet, err := q.generatorRing.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, fmt.Errorf("error finding generators in Querier.queryRangeRecent: %w", err)
	}

	// correct max series limit logic should've been set by the query-frontend sharder
	c, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeSum, int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	results, err := q.forGivenGenerators(ctx, replicationSet, func(ctx context.Context, client tempopb.MetricsGeneratorClient) (any, error) {
		return client.QueryRange(ctx, req)
	})
	if err != nil {
		_ = level.Error(log.Logger).Log("msg", "error querying generators in Querier.queryRangeRecent", "err", err)
		return nil, fmt.Errorf("error querying generators in Querier.queryRangeRecent: %w", err)
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

func (q *Querier) queryBlock(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.queryBlock: %w", err)
	}

	blockID, err := backend.ParseUUID(req.BlockID)
	if err != nil {
		return nil, err
	}

	enc, err := backend.ParseEncoding(req.Encoding)
	if err != nil {
		return nil, err
	}

	dc, err := backend.DedicatedColumnsFromTempopb(req.DedicatedColumns)
	if err != nil {
		return nil, err
	}

	meta := &backend.BlockMeta{
		Version:   req.Version,
		TenantID:  tenantID,
		StartTime: time.Unix(0, int64(req.Start)),
		EndTime:   time.Unix(0, int64(req.End)),
		Encoding:  enc,
		// IndexPageSize:    req.IndexPageSize,
		// TotalRecords:     req.TotalRecords,
		BlockID: blockID,
		// DataEncoding:     req.DataEncoding,
		Size_:            req.Size_,
		FooterSize:       req.FooterSize,
		DedicatedColumns: dc,
	}

	opts := common.DefaultSearchOptions()
	opts.StartPage = int(req.StartPage)
	opts.TotalPages = int(req.PagesToSearch)

	unsafe := q.limits.UnsafeQueryHints(tenantID)

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return nil, err
	}

	timeOverlapCutoff := q.cfg.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}

	eval, err := traceql.NewEngine().CompileMetricsQueryRange(req, int(req.Exemplars), timeOverlapCutoff, unsafe)
	if err != nil {
		return nil, err
	}

	f := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return q.store.Fetch(ctx, meta, req, opts)
	})
	err = eval.Do(ctx, f, uint64(meta.StartTime.UnixNano()), uint64(meta.EndTime.UnixNano()), int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	res := eval.Results()

	inspectedBytes, spansTotal, _ := eval.Metrics()

	if req.MaxSeries > 0 && len(res) > int(req.MaxSeries) {
		limitedRes := make(traceql.SeriesSet)
		count := 0
		for k, v := range res {
			if count >= int(req.MaxSeries) {
				break
			}
			limitedRes[k] = v
			count++
		}
		res = limitedRes
	}

	response := &tempopb.QueryRangeResponse{
		Series: res.ToProto(req),
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: inspectedBytes,
			InspectedSpans: spansTotal,
		},
	}

	if req.MaxSeries > 0 && len(res) > int(req.MaxSeries) {
		response.Status = tempopb.PartialStatus_PARTIAL
	}

	return response, nil
}
