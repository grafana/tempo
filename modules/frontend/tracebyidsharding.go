package frontend

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto" //nolint:all //deprecated
	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
)

const (
	minQueryShards = 2
	maxQueryShards = 100_000
)

func newTraceByIDSharder(cfg *TraceByIDConfig, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return shardQuery{
			next:            next,
			cfg:             cfg,
			logger:          logger,
			blockBoundaries: createBlockBoundaries(cfg.QueryShards - 1), // one shard will be used to query ingesters
		}
	})
}

type shardQuery struct {
	next            http.RoundTripper
	cfg             *TraceByIDConfig
	logger          log.Logger
	blockBoundaries [][]byte
}

// RoundTrip implements http.RoundTripper
func (s shardQuery) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardQuery")
	defer span.Finish()

	_, err := user.ExtractOrgID(ctx)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}, nil
	}

	// context propagation
	r = r.WithContext(ctx)
	reqs, err := s.buildShardedRequests(r)
	if err != nil {
		return nil, err
	}

	// execute requests
	concurrentShards := uint(s.cfg.QueryShards)
	if s.cfg.ConcurrentShards > 0 {
		concurrentShards = uint(s.cfg.ConcurrentShards)
	}
	wg := boundedwaitgroup.New(concurrentShards)
	mtx := sync.Mutex{}

	var overallError error
	combiner := trace.NewCombiner()
	combiner.Consume(&tempopb.Trace{}) // The query path returns a non-nil result even if no inputs (which is different than other paths which return nil for no inputs)
	statusCode := http.StatusNotFound
	statusMsg := "trace not found"

	for _, req := range reqs {
		wg.Add(1)
		go func(innerR *http.Request) {
			defer wg.Done()

			resp, err := s.next.RoundTrip(innerR)

			mtx.Lock()
			defer mtx.Unlock()
			if err != nil {
				overallError = err
			}

			if shouldQuit(r.Context(), statusCode, overallError) {
				return
			}

			// check http error
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error querying proxy target", "url", innerR.RequestURI, "err", err)
				overallError = err
				return
			}

			// if the status code is anything but happy, save the error and pass it down the line
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
				// todo: if we cancel the parent context here will it shortcircuit the other queries and fail fast?
				statusCode = resp.StatusCode
				bytesMsg, err := io.ReadAll(resp.Body)
				if err != nil {
					_ = level.Error(s.logger).Log("msg", "error reading response body status != ok", "url", innerR.RequestURI, "err", err)
				}
				statusMsg = string(bytesMsg)
				return
			}

			// read the body
			buff, err := io.ReadAll(resp.Body)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error reading response body status == ok", "url", innerR.RequestURI, "err", err)
				overallError = err
				return
			}

			// marshal into a trace to combine.
			// todo: better define responsibilities between middleware. the parent middleware in frontend.go actually sets the header
			//  which forces the body here to be a proto encoded tempopb.Trace{}
			traceResp := &tempopb.TraceByIDResponse{}
			err = proto.Unmarshal(buff, traceResp)
			if err != nil {
				_ = level.Error(s.logger).Log("msg", "error unmarshalling response", "url", innerR.RequestURI, "err", err, "body", string(buff))
				overallError = err
				return
			}

			// if not found bail
			if resp.StatusCode == http.StatusNotFound {
				return
			}

			// happy path
			statusCode = http.StatusOK
			combiner.Consume(traceResp.Trace)
		}(req)
	}
	wg.Wait()

	if overallError != nil {
		return nil, overallError
	}

	overallTrace, _ := combiner.Result()
	if overallTrace == nil || statusCode != http.StatusOK {
		// translate non-404s into 500s. if, for instance, we get a 400 back from an internal component
		// it means that we created a bad request. 400 should not be propagated back to the user b/c
		// the bad request was due to a bug on our side, so return 500 instead.
		if statusCode != http.StatusNotFound {
			statusCode = 500
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
func (s *shardQuery) buildShardedRequests(parent *http.Request) ([]*http.Request, error) {
	ctx := parent.Context()
	userID, err := user.ExtractOrgID(ctx)
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

		reqs[i].Header.Set(user.OrgIDHeaderName, userID)
		uri := buildUpstreamRequestURI(reqs[i].URL.Path, q)
		reqs[i].RequestURI = uri
	}

	return reqs, nil
}

// createBlockBoundaries splits the range of blockIDs into queryShards parts
func createBlockBoundaries(queryShards int) [][]byte {
	if queryShards == 0 {
		return nil
	}

	// create sharded queries
	blockBoundaries := make([][]byte, queryShards+1)
	for i := 0; i < queryShards+1; i++ {
		blockBoundaries[i] = make([]byte, 16)
	}

	// bucketSz is the min size for the bucket
	bucketSz := (math.MaxUint64 / uint64(queryShards))
	// numLarger is the number of buckets that have to be bumped by 1
	numLarger := (math.MaxUint64 % uint64(queryShards))
	boundary := uint64(0)
	for i := 0; i < queryShards; i++ {
		binary.BigEndian.PutUint64(blockBoundaries[i][:8], boundary)
		binary.BigEndian.PutUint64(blockBoundaries[i][8:], 0)

		boundary += bucketSz
		if numLarger != 0 {
			numLarger--
			boundary++
		}
	}

	binary.BigEndian.PutUint64(blockBoundaries[queryShards][:8], math.MaxUint64)
	binary.BigEndian.PutUint64(blockBoundaries[queryShards][8:], math.MaxUint64)

	return blockBoundaries
}

func shouldQuit(ctx context.Context, statusCode int, err error) bool {
	if err != nil {
		return true
	}
	if ctx.Err() != nil {
		return true
	}
	if statusCode/100 == 5 { // bail on any 5xx's
		return true
	}

	return false
}
