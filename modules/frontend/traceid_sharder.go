package frontend

import (
	"encoding/hex"
	"math"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all //deprecated
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/pkg/validation"
	"github.com/grafana/tempo/tempodb"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	minQueryShards = 2
	maxQueryShards = 100_000
)

type asyncTraceSharder struct {
	next                  pipeline.AsyncRoundTripper[combiner.PipelineResponse]
	cfg                   *TraceByIDConfig
	reader                tempodb.Reader
	logger                log.Logger
	blockBoundaries       [][]byte
	jobsPerQuery          *prometheus.HistogramVec
	maxDynamicBlockShards int // 0 means uncapped
}

func newAsyncTraceIDSharder(cfg *TraceByIDConfig, maxOutstandingPerTenant int, reader tempodb.Reader, jobsPerQuery *prometheus.HistogramVec, logger log.Logger) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		// Fixed jobs that are always emitted: 1 ingester (+ 1 external when enabled).
		fixedJobs := 1
		if cfg.ExternalEnabled {
			fixedJobs++
		}

		// Pre-computed boundaries for the query_shards path (used when blocks_per_shard isn't configured).
		fixedBlockShards := cfg.QueryShards - fixedJobs
		blockBoundaries := blockboundary.CreateBlockBoundaries(fixedBlockShards)

		// Set the maximum for the dynamic blocks_per_shard path, which can never exceed
		// the max number of jobs per tenant (minus the fixed jobs).
		maxDynamicBlockShards := 0
		if maxOutstandingPerTenant > 0 {
			maxDynamicBlockShards = maxOutstandingPerTenant - fixedJobs
		}

		return asyncTraceSharder{
			next:                  next,
			cfg:                   cfg,
			reader:                reader,
			logger:                logger,
			blockBoundaries:       blockBoundaries,
			jobsPerQuery:          jobsPerQuery,
			maxDynamicBlockShards: maxDynamicBlockShards,
		}
	})
}

// RoundTrip implements http.RoundTripper
func (s asyncTraceSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	ctx, span := tracer.Start(pipelineRequest.Context(), "frontend.ShardQuery")
	defer span.End()
	pipelineRequest.SetContext(ctx)

	_, _, _, startTime, endTime, err := api.ParseTraceByIDRequest(pipelineRequest.HTTPRequest())
	if err != nil {
		return pipeline.NewBadRequest(err), nil
	}

	reqs, err := s.buildShardedRequests(pipelineRequest, startTime, endTime)
	if err != nil {
		return nil, err
	}

	shards := len(reqs)
	s.jobsPerQuery.WithLabelValues(traceByIDOp).Observe(float64(shards))

	// execute requests
	concurrentShards := uint(shards)
	// if concurrent shards is set, respect that value
	if s.cfg.ConcurrentShards > 0 {
		concurrentShards = uint(s.cfg.ConcurrentShards)
	}

	// concurrent_shards greater than totalShards should not be allowed because it would create
	// more goroutines than the jobs to send to queriers.
	if concurrentShards > uint(shards) {
		concurrentShards = uint(shards)
	}

	return pipeline.NewAsyncSharderFunc(ctx, int(concurrentShards), len(reqs), func(i int) pipeline.Request {
		pipelineReq := reqs[i]
		return pipelineReq
	}, s.next), nil
}

// blockBoundariesForTenant returns the block boundaries to use when building sharded requests.
//
// When cfg.BlocksPerShard > 0 the number of block shards is derived dynamically from
// the current blocklist length: numBlockShards = ceil(len(blocklist) / BlocksPerShard).
// If startTime and endTime are non-zero, only blocks that overlap that time range are
// counted, giving a more accurate shard estimate for time-bounded queries.
// Otherwise the pre-computed boundaries (based on QueryShards) are returned unchanged.
func (s *asyncTraceSharder) blockBoundariesForTenant(tenantID string, startTime, endTime time.Time) [][]byte {
	if s.cfg.BlocksPerShard == 0 {
		return s.blockBoundaries
	}

	blocks := s.reader.BlockMetas(tenantID)

	// If time range is provided, filter it down to the blocks within range. We
	// don't care about the actual blocks here, it's just an estimate to determine
	// the number of shards.
	if !startTime.IsZero() && !endTime.IsZero() {
		blocks = blockMetasForSearch(blocks, startTime, endTime, acceptAllBlocks)
	}

	numBlockShards := int(math.Ceil(float64(len(blocks)) / float64(s.cfg.BlocksPerShard)))

	// Always ensure at least one job is created.
	// Even in case the query-frontend sees zero blocks within range, it's
	// possible that the querier may have a different view of the blocklist.
	if numBlockShards < 1 {
		numBlockShards = 1
	}

	// Never exceed the max queue depth (0 means uncapped).
	// It is better to run with sub-optimal sharding, than exceed the max queue depth and fail.
	if s.maxDynamicBlockShards > 0 && numBlockShards > s.maxDynamicBlockShards {
		numBlockShards = s.maxDynamicBlockShards
	}

	return blockboundary.CreateBlockBoundaries(numBlockShards)
}

// buildShardedRequests returns a slice of requests sharded on block boundaries.
// When cfg.BlocksPerShard > 0 the boundaries are computed from the live blocklist;
// otherwise the pre-computed boundaries (derived from query_shards) are used.
func (s *asyncTraceSharder) buildShardedRequests(parent pipeline.Request, startTime, endTime time.Time) ([]pipeline.Request, error) {
	userID, err := validation.ExtractValidTenantID(parent.Context())
	if err != nil {
		return nil, err
	}

	blockBoundaries := s.blockBoundariesForTenant(userID, startTime, endTime)

	reqs := make([]pipeline.Request, 0, len(blockBoundaries))
	params := map[string]string{}

	// Job 0: ingester job
	req, err := cloneRequestforQueriers(parent, userID, func(r *http.Request) (*http.Request, error) {
		params[querier.QueryModeKey] = querier.QueryModeIngesters
		return api.BuildQueryRequest(r, params), nil
	})
	if err != nil {
		return nil, err
	}
	reqs = append(reqs, req)

	// Job 1: external job (if enabled)
	if s.cfg.ExternalEnabled {
		req, err = cloneRequestforQueriers(parent, userID, func(r *http.Request) (*http.Request, error) {
			params[querier.QueryModeKey] = querier.QueryModeExternal
			return api.BuildQueryRequest(r, params), nil
		})
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}

	// Jobs 2 to N-1: block queries
	// When external is enabled, we have N-2 block shards
	// When external is disabled, we have N-1 block shards
	// blockBoundaries has length equal to numBlockShards+1, and we create shards between adjacent boundaries
	for i := 1; i < len(blockBoundaries); i++ {
		pipelineR, _ := cloneRequestforQueriers(parent, userID, func(r *http.Request) (*http.Request, error) {
			// block queries
			params[querier.BlockStartKey] = hex.EncodeToString(blockBoundaries[i-1])
			params[querier.BlockEndKey] = hex.EncodeToString(blockBoundaries[i])
			params[querier.QueryModeKey] = querier.QueryModeBlocks

			return api.BuildQueryRequest(r, params), nil
		})
		reqs = append(reqs, pipelineR)
	}

	return reqs, nil
}
