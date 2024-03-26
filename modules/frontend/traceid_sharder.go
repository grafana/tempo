package frontend

import (
	"context"
	"encoding/hex"
	"net/http"

	"github.com/go-kit/log" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/blockboundary"
)

const (
	minQueryShards = 2
	maxQueryShards = 100_000
)

type asyncTraceSharder struct {
	next            pipeline.AsyncRoundTripper[*http.Response]
	cfg             *TraceByIDConfig
	logger          log.Logger
	blockBoundaries [][]byte
}

func newAsyncTraceIDSharder(cfg *TraceByIDConfig, logger log.Logger) pipeline.AsyncMiddleware[*http.Response] {
	return pipeline.AsyncMiddlewareFunc[*http.Response](func(next pipeline.AsyncRoundTripper[*http.Response]) pipeline.AsyncRoundTripper[*http.Response] {
		return asyncTraceSharder{
			next:            next,
			cfg:             cfg,
			logger:          logger,
			blockBoundaries: blockboundary.CreateBlockBoundaries(cfg.QueryShards - 1), // one shard will be used to query ingesters
		}
	})
}

// RoundTrip implements http.RoundTripper
func (s asyncTraceSharder) RoundTrip(r *http.Request) (pipeline.Responses[*http.Response], error) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "frontend.ShardQuery")
	defer span.Finish()
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

	return pipeline.NewAsyncSharderFunc(ctx, int(concurrentShards), len(reqs), func(i int) *http.Request {
		return reqs[i]
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
	// build sharded block queries
	for i := 0; i < len(s.blockBoundaries); i++ {
		reqs[i] = parent.Clone(ctx)

		q := reqs[i].URL.Query()
		if i == 0 {
			// ingester query
			q.Add(querier.QueryModeKey, querier.QueryModeIngesters)
		} else {
			// block queries
			q.Add(querier.BlockStartKey, hex.EncodeToString(s.blockBoundaries[i-1]))
			q.Add(querier.BlockEndKey, hex.EncodeToString(s.blockBoundaries[i]))
			q.Add(querier.QueryModeKey, querier.QueryModeBlocks)
		}

		prepareRequestForDownstream(reqs[i], userID, reqs[i].URL.Path, q)
	}

	return reqs, nil
}
