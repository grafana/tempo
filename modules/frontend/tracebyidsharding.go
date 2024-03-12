package frontend

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/opentracing/opentracing-go"

	"github.com/grafana/tempo/modules/frontend/pipeline"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	minQueryShards = 2
	maxQueryShards = 100_000
)

func newTraceByIDSharder(cfg *TraceByIDConfig, o overrides.Interface, logger log.Logger) pipeline.Middleware {
	return pipeline.MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return shardQuery{
			next:            next,
			cfg:             cfg,
			logger:          logger,
			o:               o,
			blockBoundaries: blockboundary.CreateBlockBoundaries(cfg.QueryShards - 1), // one shard will be used to query ingesters
		}
	})
}

type shardQuery struct {
	next            http.RoundTripper
	cfg             *TraceByIDConfig
	logger          log.Logger
	o               overrides.Interface
	blockBoundaries [][]byte
}

// RoundTrip implements http.RoundTripper
func (s shardQuery) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardQuery")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	// context propagation
	r = r.WithContext(ctx)
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	reqs, err := s.buildShardedRequests(subCtx, r)
	if err != nil {
		return nil, err
	}

	// execute requests
	concurrentShards := uint(s.cfg.QueryShards)
	if s.cfg.ConcurrentShards > 0 {
		concurrentShards = uint(s.cfg.ConcurrentShards)
	}

	var (
		overallError error

		mtx        = sync.Mutex{}
		statusCode = http.StatusNotFound
		statusMsg  = "trace not found"
		wg         = boundedwaitgroup.New(concurrentShards)
	)

	combiner := trace.NewCombiner(s.o.MaxBytesPerTrace(userID))
	_, _ = combiner.Consume(&tempopb.Trace{}) // The query path returns a non-nil result even if no inputs (which is different than other paths which return nil for no inputs)

	for _, req := range reqs {
		wg.Add(1)
		go func(innerR *http.Request) {
			defer wg.Done()

			resp, rtErr := s.next.RoundTrip(innerR)

			mtx.Lock()
			defer mtx.Unlock()
			if rtErr != nil {
				overallError = rtErr
			}

			// Check the context of the worker request
			if shouldQuit(innerR.Context(), statusCode, overallError) {
				return
			}

			// if the status code is anything but happy, save the error and pass it
			// down the line
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
				defer subCancel()

				statusCode = resp.StatusCode
				bytesMsg, readErr := io.ReadAll(resp.Body)
				if readErr != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", readErr)
				}
				statusMsg = string(bytesMsg)
				return
			}

			// read the body
			buff, rtErr := io.ReadAll(resp.Body)
			if rtErr != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", rtErr)
				overallError = rtErr
				return
			}

			// marshal into a trace to combine.
			// TODO: better define responsibilities between middleware. the parent middleware in frontend.go actually sets the header
			//  which forces the body here to be a proto encoded tempopb.Trace{}
			traceResp := &tempopb.TraceByIDResponse{}
			rtErr = proto.Unmarshal(buff, traceResp)
			if rtErr != nil {
				_ = level.Error(s.logger).Log("msg", "error unmarshalling response", "url", innerR.RequestURI, "err", rtErr, "body", string(buff))
				overallError = rtErr
				return
			}

			// if not found bail
			if resp.StatusCode == http.StatusNotFound {
				return
			}

			// happy path
			statusCode = http.StatusOK
			_, rtErr = combiner.Consume(traceResp.Trace)
			if rtErr != nil {
				overallError = rtErr
			}
		}(req)
	}
	wg.Wait()

	if overallError != nil {
		return nil, overallError
	}

	overallTrace, _ := combiner.Result()
	if overallTrace == nil || statusCode != http.StatusOK {
		// TODO: reevaluate - should we propagate 400's back to the user?
		// translate non-404s into 500s. if, for instance, we get a 400 back from an internal component
		// it means that we created a bad request. 400 should not be propagated back to the user b/c
		// the bad request was due to a bug on our side, so return 500 instead.

		switch statusCode {
		case http.StatusNotFound:
			// Pass through 404s
		case http.StatusTooManyRequests:
			// Pass through 429s
		default:
			statusCode = http.StatusInternalServerError
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(statusMsg)),
			Header:     http.Header{},
		}, nil
	}

	buff, err := proto.Marshal(&tempopb.TraceByIDResponse{
		Trace:   overallTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	})
	if err != nil {
		_ = level.Error(s.logger).Log("msg", "error marshalling response to proto", "err", err)
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptProtobuf},
		},
		Body:          io.NopCloser(bytes.NewReader(buff)),
		ContentLength: int64(len(buff)),
	}, nil
}

// buildShardedRequests returns a slice of requests sharded on the precalculated
// block boundaries
func (s *shardQuery) buildShardedRequests(ctx context.Context, parent *http.Request) ([]*http.Request, error) {
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

func shouldQuit(ctx context.Context, statusCode int, err error) bool {
	if err != nil {
		return true
	}

	if ctx.Err() != nil {
		return true
	}

	if statusCode == http.StatusTooManyRequests {
		return true
	}

	return statusCode/100 == 5 // bail on any 5xx's
}
