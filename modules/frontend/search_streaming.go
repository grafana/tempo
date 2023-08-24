package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

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

// jpe slos on this?
// newSearchStreamingHandler returns a handler that streams results from the HTTP handler
func newSearchStreamingHandler(cfg Config, o overrides.Interface, downstream http.RoundTripper, reader tempodb.Reader, apiPrefix string, logger log.Logger) streamingSearchHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearch)
	return func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
		// build search request and propagate context
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
		ctx := srv.Context()
		httpReq = httpReq.WithContext(ctx)

		// streaming search only accepts requests with backend components
		if !api.IsBackendSearch(httpReq) {
			level.Error(logger).Log("msg", "search streaming: start/end date not provided")
			return errors.New("request must contain a start/end date for streaming search")
		}

		progress := atomic.NewPointer[*diffSearchProgress](nil)
		fn := func(ctx context.Context, limit, totalJobs, totalBlocks int, totalBlockBytes uint64) shardedSearchProgress {
			p := newDiffSearchProgress(ctx, limit, totalJobs, totalBlocks, totalBlockBytes)
			progress.Store(&p)
			return p
		}
		// build roundtripper
		rt := NewRoundTripper(downstream, newSearchSharder(reader, o, cfg.Search.Sharder, fn, logger))

		type roundTripResult struct {
			resp *http.Response
			err  error
		}
		resultChan := make(chan roundTripResult)

		// initiate http pipeline
		go func() {
			resp, err := rt.RoundTrip(httpReq)
			resultChan <- roundTripResult{resp, err}
			close(resultChan)
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

				err = srv.Send(result.response)
				if err != nil {
					level.Error(logger).Log("msg", "search streaming: send failed", "err", err)
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

					level.Error(logger).Log("msg", "search streaming: status != 200", "status", roundTripRes.resp.StatusCode, "body", string(b))
					return fmt.Errorf("http error: %d msg: %s", roundTripRes.resp.StatusCode, string(b))
				}

				// overall pipeline returned successfully, now grab the final results and send them
				p := *progress.Load()
				result := p.finalResult()
				if result.err != nil || result.statusCode != http.StatusOK {
					level.Error(logger).Log("msg", "search streaming: result status != 200", "err", result.err, "status", result.statusCode, "body", result.statusMsg)
					return fmt.Errorf("result error: %d status: %d msg: %s", result.err, result.statusCode, result.statusMsg)
				}
				err = srv.Send(result.response)
				if err != nil {
					level.Error(logger).Log("msg", "search streaming: send failed", "err", err)
					return fmt.Errorf("search streaming send failed: %w", err)
				}

				return nil
			}
		}
	}
}
