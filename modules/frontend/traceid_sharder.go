package frontend

import (
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-kit/log" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/frontend/combiner"
	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/blockboundary"
)

const (
	minQueryShards = 2
	maxQueryShards = 100_000
)

type asyncTraceSharder struct {
	next            pipeline.AsyncRoundTripper[combiner.PipelineResponse]
	cfg             *TraceByIDConfig
	logger          log.Logger
	blockBoundaries [][]byte
}

func newAsyncTraceIDSharder(cfg *TraceByIDConfig, logger log.Logger) pipeline.AsyncMiddleware[combiner.PipelineResponse] {
	return pipeline.AsyncMiddlewareFunc[combiner.PipelineResponse](func(next pipeline.AsyncRoundTripper[combiner.PipelineResponse]) pipeline.AsyncRoundTripper[combiner.PipelineResponse] {
		return asyncTraceSharder{
			next:            next,
			cfg:             cfg,
			logger:          logger,
			blockBoundaries: blockboundary.CreateBlockBoundaries(cfg.QueryShards - 1), // one shard will be used to query ingesters
		}
	})
}

// RoundTrip implements http.RoundTripper
func (s asyncTraceSharder) RoundTrip(pipelineRequest pipeline.Request) (pipeline.Responses[combiner.PipelineResponse], error) {
	ctx, span := tracer.Start(pipelineRequest.Context(), "frontend.ShardQuery")
	defer span.End()
	pipelineRequest.SetContext(ctx)

	reqs, err := s.buildShardedRequests(pipelineRequest)
	if err != nil {
		return nil, err
	}

	// execute requests
	concurrentShards := uint(s.cfg.QueryShards)
	// if concurrent shards is set, respect that value
	if s.cfg.ConcurrentShards > 0 {
		concurrentShards = uint(s.cfg.ConcurrentShards)
	}

	// concurrent_shards grater then query_shards should not be allowed because it would create
	// more goroutines then the jobs to send these jobs to queriers.
	if concurrentShards > uint(s.cfg.QueryShards) {
		// set the concurrent shards to the total shards
		concurrentShards = uint(s.cfg.QueryShards)
	}

	return pipeline.NewAsyncSharderFunc(ctx, int(concurrentShards), len(reqs), func(i int) pipeline.Request {
		pipelineReq := reqs[i]
		return pipelineReq
	}, s.next), nil
}

// buildShardedRequests returns a slice of requests sharded on the precalculated
// block boundaries
func (s *asyncTraceSharder) buildShardedRequests(parent pipeline.Request) ([]pipeline.Request, error) {
	userID, err := user.ExtractOrgID(parent.Context())
	if err != nil {
		return nil, err
	}

	reqs := make([]pipeline.Request, s.cfg.QueryShards)
	params := map[string]string{}

	reqs[0], err = cloneRequestforQueriers(parent, userID, func(r *http.Request) (*http.Request, error) {
		params[querier.QueryModeKey] = querier.QueryModeIngesters
		return api.BuildQueryRequest(r, params), nil
	})
	if err != nil {
		return nil, err
	}

	var rf1After string
	if val := parent.HTTPRequest().URL.Query().Get(api.URLParamRF1After); val != "" {
		rf1After = val
	} else if !s.cfg.RF1After.IsZero() {
		rf1After = s.cfg.RF1After.Format(time.RFC3339)
	}

	// build sharded block queries
	for i := 1; i < len(s.blockBoundaries); i++ {
		i := i // save the loop variable locally to make sure the closure grabs the correct var.
		pipelineR, _ := cloneRequestforQueriers(parent, userID, func(r *http.Request) (*http.Request, error) {
			// block queries
			params[querier.BlockStartKey] = hex.EncodeToString(s.blockBoundaries[i-1])
			params[querier.BlockEndKey] = hex.EncodeToString(s.blockBoundaries[i])
			params[querier.QueryModeKey] = querier.QueryModeBlocks
			params[api.URLParamRF1After] = rf1After

			return api.BuildQueryRequest(r, params), nil
		})

		reqs[i] = pipelineR
	}

	return reqs, nil
}
