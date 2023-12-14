package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	//nolint:all //deprecated
	"github.com/grafana/tempo/modules/frontend/combiner"
	"go.uber.org/atomic"

	"github.com/grafana/dskit/user"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
)

// diffSearchProgress only returns new and updated traces when result() is called
// it uses a wrapped searchProgress to do all of the normal tracking as well as a map
// to track if a trace was updated or not
type diffSearchProgress struct {
	progress shardedSearchProgress

	seenTraces map[string]struct{}
	mtx        sync.Mutex
}

func newDiffSearchProgress(ctx context.Context, limit, totalJobs, totalBlocks int, totalBlockBytes uint64) *diffSearchProgress {
	return &diffSearchProgress{
		seenTraces: map[string]struct{}{},
		progress:   newSearchProgress(ctx, limit, totalJobs, totalBlocks, totalBlockBytes),
	}
}

func (p *diffSearchProgress) setStatus(statusCode int, statusMsg string) {
	p.progress.setStatus(statusCode, statusMsg)
}

func (p *diffSearchProgress) setError(err error) {
	p.progress.setError(err)
}

func (p *diffSearchProgress) addResponse(res *tempopb.SearchResponse) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// record modified traces
	for _, trace := range res.Traces {
		p.seenTraces[trace.TraceID] = struct{}{}
	}
	p.progress.addResponse(res)
}

// shouldQuit locks and checks if we should quit from current execution or not
func (p *diffSearchProgress) shouldQuit() bool {
	return p.progress.shouldQuit()
}

func (p *diffSearchProgress) result() *shardedSearchResults {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	res := p.progress.result()

	// filter result down to only traces in seenTraces
	if res.response != nil {
		keepTraces := make([]*tempopb.TraceSearchMetadata, 0, len(res.response.Traces))
		for _, trace := range res.response.Traces {
			_, ok := p.seenTraces[trace.TraceID]
			if ok {
				keepTraces = append(keepTraces, trace)
			}
		}
		res.response.Traces = keepTraces
	}
	// clear seen traces
	p.seenTraces = map[string]struct{}{}

	return res
}

// finalResult gives the user the ability to pull all results w/o filtering
// to ensure that all results are sent to the caller
func (p *diffSearchProgress) finalResult() *shardedSearchResults {
	return p.progress.result()
}

// newSearchStreamingGRPCHandler returns a handler that streams results from the HTTP handler
func newSearchStreamingGRPCHandler(cfg Config, o overrides.Interface, downstream http.RoundTripper, reader tempodb.Reader, searchCache *frontendCache, apiPrefix string, logger log.Logger) streamingSearchHandler {
	searcher := streamingSearcher{
		logger:        logger,
		downstream:    downstream,
		reader:        reader,
		postSLOHook:   searchSLOPostHook(cfg.Search.SLO),
		o:             o,
		searchCache:   searchCache,
		cfg:           &cfg,
		preMiddleware: newMultiTenantMiddleware(cfg, combiner.NewSearch, logger),
	}

	downstreamPath := path.Join(apiPrefix, api.PathSearch)
	return func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
		// tenant, ctx, err := user.ExtractFromGRPCRequest(srv.Context())
		// if err != nil {
		// 	level.Error(logger).Log("msg", "search streaming: extract org id failed", "err", err)
		// 	return fmt.Errorf("extract org id failed: %w", err)
		// }
		//
		// level.Debug(logger).Log("msg", "in search streaming", "tenant", tenant)

		httpReq, err := api.BuildSearchRequest(&http.Request{
			URL: &url.URL{
				Path: downstreamPath,
			},
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
			RequestURI: buildUpstreamRequestURI(downstreamPath, nil),
		}, req)
		if err != nil {
			level.Error(logger).Log("msg", "search streaming: build search request failed", "err", err)
			return fmt.Errorf("build search request failed: %w", err)
		}

		// we inject context here?? why??
		httpReq = httpReq.WithContext(srv.Context())

		return searcher.handle(httpReq, func(resp *tempopb.SearchResponse) error {
			return srv.Send(resp)
		})
	}
}

type streamingSearcher struct {
	logger        log.Logger
	downstream    http.RoundTripper
	reader        tempodb.Reader
	postSLOHook   handlerPostHook
	o             overrides.Interface
	searchCache   *frontendCache
	cfg           *Config
	preMiddleware Middleware
}

// multi-tenant support in handle func, and that will make the streaming for multi-tenant search work??
func (s *streamingSearcher) handle(r *http.Request, forwardResults func(*tempopb.SearchResponse) error) error {
	ctx := r.Context()

	// SLOS - start timer and prep context
	start := time.Now()
	tenant, _ := user.ExtractOrgID(ctx)
	ctx = searchSLOPreHook(ctx)

	// streaming search only accepts requests with backend components
	if !api.IsBackendSearch(r) {
		return errors.New("request must contain a start/end date for streaming search")
	}

	progress := atomic.NewPointer[*diffSearchProgress](nil)
	fn := func(ctx context.Context, limit, totalJobs, totalBlocks int, totalBlockBytes uint64) shardedSearchProgress {
		// FIXME: we need to use a single diff searcher across all tenants??
		//  or maybe a searcher that sums up things across multi-tenants??
		p := newDiffSearchProgress(ctx, limit, totalJobs, totalBlocks, totalBlockBytes)
		progress.Store(&p)
		return p
	}

	// TODO: we passed multi-tenant middleware as downstream, and it is pre-fixed before our
	// search sharding middleware?? and it works because we run this before we do search sharding??
	// order of ops:
	// - newMultiTenantMiddleware
	// - newSearchSharder
	// - s.downstream

	// build roundtripper
	ss := newSearchSharder(s.reader, s.o, s.cfg.Search.Sharder, fn, s.searchCache, s.logger)

	// preMiddleware will make multiple calls to newSearchSharder in case of multi-tenant search
	// and it will end up creating it's own newDiffSearchProgress for each tenant?
	// we need to make fn add up the values instead of giving new values each time?
	rt := NewRoundTripper(s.downstream, s.preMiddleware, ss)

	type roundTripResult struct {
		resp *http.Response
		err  error
	}
	resultChan := make(chan roundTripResult)

	// initiate http pipeline
	go func() {
		resp, err := rt.RoundTrip(r)
		resultChan <- roundTripResult{resp, err}
		close(resultChan)

		// SLOs record results
		s.postSLOHook(ctx, resp, tenant, time.Since(start), err)
	}()

	// collect and return results
	for {
		select {
		// handles context canceled or other errors
		case <-ctx.Done():
			return ctx.Err()
		// stream results as they come in
		case <-time.After(500 * time.Millisecond):
			p := progress.Load()
			if p == nil {
				continue
			}

			result := (*p).result()
			if result.err != nil || result.statusCode != http.StatusOK { // ignore errors here, we'll get them in the resultChan
				continue
			}

			err := forwardResults(result.response)
			if err != nil {
				level.Error(s.logger).Log("msg", "search streaming: send failed", "err", err)
				return fmt.Errorf("search streaming send failed: %w", err)
			}
		// final result is available
		case roundTripRes := <-resultChan:
			// check for errors in the http response
			if roundTripRes.err != nil {
				return roundTripRes.err
			}
			if roundTripRes.resp != nil && roundTripRes.resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(roundTripRes.resp.Body)

				level.Error(s.logger).Log("msg", "search streaming: status != 200", "status", roundTripRes.resp.StatusCode, "body", string(b))
				return fmt.Errorf("http error: %d msg: %s", roundTripRes.resp.StatusCode, string(b))
			}

			// overall pipeline returned successfully, now grab the final results and send them
			p := *progress.Load()
			result := p.finalResult()
			if result.err != nil || result.statusCode != http.StatusOK {
				level.Error(s.logger).Log("msg", "search streaming: result status != 200", "err", result.err, "status", result.statusCode, "body", result.statusMsg)
				return fmt.Errorf("result error: %d status: %d msg: %s", result.err, result.statusCode, result.statusMsg)
			}
			err := forwardResults(result.response)
			if err != nil {
				level.Error(s.logger).Log("msg", "search streaming: send failed", "err", err)
				return fmt.Errorf("search streaming send failed: %w", err)
			}

			return nil
		}
	}
}
