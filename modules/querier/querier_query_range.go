package querier

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
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
	// // Get results from all generators
	replicationSet, err := q.generatorRing.GetReplicationSetForOperation(ring.Read)
	if err != nil {
		return nil, fmt.Errorf("error finding generators in Querier.SpanMetricsSummary: %w", err)
	}
	lookupResults, err := q.forGivenGenerators(
		ctx,
		replicationSet,
		func(ctx context.Context, client tempopb.MetricsGeneratorClient) (interface{}, error) {
			return client.QueryRange(ctx, req)
		},
	)
	if err != nil {
		_ = level.Error(log.Logger).Log("error querying generators in Querier.MetricsQueryRange", "err", err)

		return nil, fmt.Errorf("error querying generators in Querier.MetricsQueryRange: %w", err)
	}

	c, err := traceql.QueryRangeCombinerFor(req, traceql.AggregateModeSum, false)
	if err != nil {
		return nil, err
	}

	for _, result := range lookupResults {
		c.Combine(result.response.(*tempopb.QueryRangeResponse))
	}

	return c.Response(), nil
}

func (q *Querier) queryBlock(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	tenantID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error extracting org id in Querier.BackendSearch: %w", err)
	}

	blockID, err := uuid.Parse(req.BlockID)
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
		Size:             req.Size_,
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
	err = eval.Do(ctx, f, uint64(meta.StartTime.UnixNano()), uint64(meta.EndTime.UnixNano()))
	if err != nil {
		return nil, err
	}

	res := eval.Results()

	inspectedBytes, spansTotal, _ := eval.Metrics()

	return &tempopb.QueryRangeResponse{
		Series: queryRangeTraceQLToProto(res, req),
		Metrics: &tempopb.SearchMetrics{
			InspectedBytes: inspectedBytes,
			InspectedSpans: spansTotal,
		},
	}, nil
}

func queryRangeTraceQLToProto(set traceql.SeriesSet, req *tempopb.QueryRangeRequest) []*tempopb.TimeSeries {
	resp := make([]*tempopb.TimeSeries, 0, len(set))

	for promLabels, s := range set {
		labels := make([]v1.KeyValue, 0, len(s.Labels))
		for _, label := range s.Labels {
			labels = append(labels,
				v1.KeyValue{
					Key:   label.Name,
					Value: label.Value.AsAnyValue(),
				},
			)
		}

		intervals := traceql.IntervalCount(req.Start, req.End, req.Step)
		samples := make([]tempopb.Sample, 0, intervals)
		for i, value := range s.Values {

			ts := traceql.TimestampOf(uint64(i), req.Start, req.Step)

			samples = append(samples, tempopb.Sample{
				TimestampMs: time.Unix(0, int64(ts)).UnixMilli(),
				Value:       value,
			})
		}

		exemplars := make([]tempopb.Exemplar, 0, len(s.Exemplars))
		for _, e := range s.Exemplars {
			lbls := make([]v1.KeyValue, 0, len(e.Labels))
			for _, label := range e.Labels {
				lbls = append(lbls,
					v1.KeyValue{
						Key:   label.Name,
						Value: label.Value.AsAnyValue(),
					},
				)
			}
			exemplars = append(exemplars, tempopb.Exemplar{
				Labels:      lbls,
				TimestampMs: int64(e.TimestampMs),
				Value:       e.Value,
			})
		}

		ss := &tempopb.TimeSeries{
			PromLabels: promLabels,
			Labels:     labels,
			Samples:    samples,
			Exemplars:  exemplars,
		}

		resp = append(resp, ss)
	}

	return resp
}
