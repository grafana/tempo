/*
 * This package is a fork of localblocks/query_range any changes should be made in both.
 */
package livestore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/segmentio/fasthash/fnv1a"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/modules/ingester"
	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	timeBuffer = 5 * time.Minute
)

// QueryRange returns metrics.
func (i *instance) QueryRange(ctx context.Context, req *tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e := traceql.NewEngine()

	// Compile the raw version of the query for head and wal blocks
	// These aren't cached and we put them all into the same evaluator
	// for efficiency.
	// TODO MRD look into how to propagate unsafe query hints.
	rawEval, err := e.CompileMetricsQueryRange(req, int(req.Exemplars), i.Cfg.Metrics.TimeOverlapCutoff, false)
	if err != nil {
		return nil, err
	}

	// This is a summation version of the query for complete blocks
	// which can be cached. They are timeseries, so they need the job-level evaluator.
	jobEval, err := traceql.NewEngine().CompileMetricsQueryRangeNonRaw(req, traceql.AggregateModeSum)
	if err != nil {
		return nil, err
	}

	i.blocksMtx.RLock()
	defer i.blocksMtx.RUnlock()

	cutoff := time.Now().Add(-i.Cfg.CompleteBlockTimeout).Add(-timeBuffer)
	if req.Start < uint64(cutoff.UnixNano()) {
		return nil, fmt.Errorf("time range must be within last %v", i.Cfg.CompleteBlockTimeout)
	}

	expr, err := traceql.Parse(req.Query)
	if err != nil {
		return nil, fmt.Errorf("compiling query: %w", err)
	}

	unsafe := i.overrides.UnsafeQueryHints(i.tenantID)

	timeOverlapCutoff := i.Cfg.Metrics.TimeOverlapCutoff
	if v, ok := expr.Hints.GetFloat(traceql.HintTimeOverlapCutoff, unsafe); ok && v >= 0 && v <= 1.0 {
		timeOverlapCutoff = v
	}

	maxSeries := int(req.MaxSeries)
	maxSeriesReached := atomic.Bool{}
	maxSeriesReached.Store(false)

	search := func(ctx context.Context, _ *backend.BlockMeta, b block) error {
		if walBlock, ok := b.(common.WALBlock); ok {
			err := i.queryRangeWALBlock(ctx, walBlock, rawEval, maxSeries)
			if err != nil {
				return err
			}
			if maxSeries > 0 && rawEval.Length() > maxSeries {
				maxSeriesReached.Store(true)
				return errComplete
			}
			return nil
		}

		if localBlock, ok := b.(*ingester.LocalBlock); ok {
			resp, err := i.queryRangeCompleteBlock(ctx, localBlock, *req, timeOverlapCutoff, unsafe, int(req.Exemplars))
			if err != nil {
				return err
			}
			jobEval.ObserveSeries(resp)
			if maxSeries > 0 && jobEval.Length() > maxSeries {
				maxSeriesReached.Store(true)
				return errComplete
			}
			return nil
		}

		return fmt.Errorf("unexpected block type: %T", b)
	}

	err = i.iterateBlocks(ctx, time.Unix(0, int64(req.Start)), time.Unix(0, int64(req.End)), search)
	if err != nil {
		level.Error(i.logger).Log("msg", "error in QueryRange", "err", err)
		return nil, err
	}

	// Combine the raw results into the job results
	walResults := rawEval.Results().ToProto(req)
	jobEval.ObserveSeries(walResults)

	r := jobEval.Results()
	rr := r.ToProto(req)

	if maxSeriesReached.Load() {
		return &tempopb.QueryRangeResponse{
			Series: rr[:maxSeries],
			Status: tempopb.PartialStatus_PARTIAL,
		}, nil
	}

	return &tempopb.QueryRangeResponse{
		Series: rr,
	}, nil
}

func (i *instance) queryRangeWALBlock(ctx context.Context, b common.WALBlock, eval *traceql.MetricsEvaluator, maxSeries int) error {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "instance.QueryRange.WALBlock", trace.WithAttributes(
		attribute.String("block", m.BlockID.String()),
		attribute.Int64("blockSize", int64(m.Size_)),
	))
	defer span.End()

	fetcher := traceql.NewSpansetFetcherWrapper(func(ctx context.Context, req traceql.FetchSpansRequest) (traceql.FetchSpansResponse, error) {
		return b.Fetch(ctx, req, common.DefaultSearchOptions())
	})

	return eval.Do(ctx, fetcher, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()), maxSeries)
}

func (i *instance) queryRangeCompleteBlock(ctx context.Context, b *ingester.LocalBlock, req tempopb.QueryRangeRequest, timeOverlapCutoff float64, unsafe bool, exemplars int) ([]*tempopb.TimeSeries, error) {
	m := b.BlockMeta()
	ctx, span := tracer.Start(ctx, "instance.QueryRange.CompleteBlock", trace.WithAttributes(
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

	cached, name, err := i.queryRangeCacheGet(ctx, m, req)
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
	err = eval.Do(ctx, f, uint64(m.StartTime.UnixNano()), uint64(m.EndTime.UnixNano()), int(req.MaxSeries))
	if err != nil {
		return nil, err
	}

	results := eval.Results().ToProto(&req)

	if name != "" {
		err = i.queryRangeCacheSet(ctx, m, name, &tempopb.QueryRangeResponse{
			Series: results,
		})
		if err != nil {
			return nil, fmt.Errorf("writing local query cache: %w", err)
		}
	}

	return results, nil
}

func (i *instance) queryRangeCacheGet(ctx context.Context, m *backend.BlockMeta, req tempopb.QueryRangeRequest) (*tempopb.QueryRangeResponse, string, error) {
	hash := queryRangeHashForBlock(req)

	name := fmt.Sprintf("cache_query_range_%v.buf", hash)

	keyPath := backend.KeyPathForBlock((uuid.UUID)(m.BlockID), m.TenantID)
	reader, size, err := i.wal.LocalBackend().Read(ctx, name, keyPath, nil)
	if err != nil {
		if errors.Is(err, backend.ErrDoesNotExist) {
			// Not cached, but return the name/keypath so it can be set after
			return nil, name, nil
		}
		return nil, "", err
	}
	defer reader.Close()

	data, err := tempo_io.ReadAllWithEstimate(reader, size)
	if err != nil {
		return nil, "", err
	}

	resp := &tempopb.QueryRangeResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return nil, "", err
	}

	return resp, name, nil
}

func (i *instance) queryRangeCacheSet(ctx context.Context, m *backend.BlockMeta, name string, resp *tempopb.QueryRangeResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	keyPath := backend.KeyPathForBlock((uuid.UUID)(m.BlockID), m.TenantID)
	return i.wal.LocalBackend().Write(ctx, name, keyPath, bytes.NewReader(data), int64(len(data)), nil)
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
