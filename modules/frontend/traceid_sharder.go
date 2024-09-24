package frontend

import (
	"context"
	"encoding/hex"
	"net/http"

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
	r := pipelineRequest.HTTPRequest()

	ctx, span := tracer.Start(r.Context(), "frontend.ShardQuery")
	defer span.End()
	r = r.WithContext(ctx)

	reqs, err := s.buildShardedRequests(ctx, r)
	if err != nil {
		return nil, err
	}

	// execute requests
	concurrentShards := uint(s.cfg.QueryShards)
	if s.cfg.ConcurrentShards > 0 {
		concurrentShards = uint(s.cfg.ConcurrentShards)
	}

	return pipeline.NewAsyncSharderFunc(ctx, int(concurrentShards), len(reqs), func(i int) pipeline.Request {
		pipelineReq := pipelineRequest.FromHTTPRequest(reqs[i])
		return pipelineReq
	}, s.next), nil
}

// buildShardedRequests returns a slice of requests sharded on the precalculated
// block boundaries
func (s *asyncTraceSharder) buildShardedRequests(ctx context.Context, parent *http.Request) ([]*http.Request, error) {
	userID, err := user.ExtractOrgID(parent.Context())
	if err != nil {
		return nil, err
	}

	reqs := make([]*http.Request, s.cfg.QueryShards)
	params := map[string]string{}
	// build sharded block queries
	for i := 0; i < len(s.blockBoundaries); i++ {
		reqs[i] = parent.Clone(ctx)
		if i == 0 {
			// ingester query
			params[querier.QueryModeKey] = querier.QueryModeIngesters
		} else {
			// block queries
			params[querier.BlockStartKey] = hex.EncodeToString(s.blockBoundaries[i-1])
			params[querier.BlockEndKey] = hex.EncodeToString(s.blockBoundaries[i])
			params[querier.QueryModeKey] = querier.QueryModeBlocks
		}
		reqs[i] = api.BuildQueryRequest(reqs[i], params)
		prepareRequestForQueriers(reqs[i], userID)
	}

	return reqs, nil
}
