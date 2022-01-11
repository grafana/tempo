package frontend

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/user"
)

const (
	minQueryShards = 2
	maxQueryShards = 256
)

func newTraceByIDSharder(queryShards, maxFailedBlocks int, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {
		return shardQuery{
			next:            next,
			queryShards:     queryShards,
			logger:          logger,
			blockBoundaries: createBlockBoundaries(queryShards - 1), // one shard will be used to query ingesters
			maxFailedBlocks: uint32(maxFailedBlocks),
		}
	})
}

type shardQuery struct {
	next            http.RoundTripper
	queryShards     int
	logger          log.Logger
	blockBoundaries [][]byte
	maxFailedBlocks uint32
}

// RoundTrip implements http.RoundTripper
func (s shardQuery) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "frontend.ShardQuery")
	defer span.Finish()

	// context propagation
	r = r.WithContext(ctx)
	reqs, err := s.buildShardedRequests(r)
	if err != nil {
		return nil, err
	}

	// execute requests
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}

	var overallError error
	var totalFailedBlocks uint32
	overallTrace := &tempopb.Trace{}
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

			if traceResp.Metrics != nil {
				totalFailedBlocks += traceResp.Metrics.FailedBlocks
				if totalFailedBlocks > s.maxFailedBlocks {
					overallError = fmt.Errorf("too many failed block queries %d (max %d)", totalFailedBlocks, s.maxFailedBlocks)
					return
				}
			}

			// if not found bail
			if resp.StatusCode == http.StatusNotFound {
				return
			}

			// happy path
			statusCode = http.StatusOK
			overallTrace, _ = trace.CombineTraceProtos(overallTrace, traceResp.Trace)
		}(req)
	}
	wg.Wait()

	if overallError != nil {
		return nil, overallError
	}

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
		Trace: overallTrace,
		Metrics: &tempopb.TraceByIDMetrics{
			FailedBlocks: totalFailedBlocks,
		},
	})
	if err != nil {
		_ = level.Error(s.logger).Log("msg", "error marshalling response to proto", "err", err)
		return nil, err
	}

	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
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

	reqs := make([]*http.Request, s.queryShards)
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
	const MaxUint = uint64(^uint8(0))
	for i := 0; i < queryShards; i++ {
		binary.LittleEndian.PutUint64(blockBoundaries[i][:8], (MaxUint/uint64(queryShards))*uint64(i))
		binary.LittleEndian.PutUint64(blockBoundaries[i][8:], 0)
	}
	const MaxUint64 = ^uint64(0)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][:8], MaxUint64)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][8:], MaxUint64)

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
