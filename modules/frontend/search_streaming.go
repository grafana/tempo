package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/pkg/errors"
)

func newSearchStreamingHandler(cfg Config, o *overrides.Overrides, downstream http.RoundTripper, reader tempodb.Reader, apiPrefix string, logger log.Logger) streamingSearchHandler {
	downstreamPath := path.Join(apiPrefix, api.PathSearch)
	return func(req *tempopb.SearchRequest, srv tempopb.StreamingQuerier_SearchServer) error {
		// build search request and propagate context
		httpReq, err := api.BuildSearchRequest(&http.Request{
			URL: &url.URL{
				Path: downstreamPath,
			},
			Header:     http.Header{},
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
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

		var p *searchProgress
		progressFn := func(ctx context.Context, limit, jobs, totalBlocks, totalBlockBytes int) shardedSearchProgress {
			p = newSearchProgress(ctx, limit, jobs, totalBlocks, totalBlockBytes).(*searchProgress)
			return p
		}

		// build roundtripper
		rt := NewRoundTripper(downstream, newSearchSharder(reader, o, cfg.Search.Sharder, cfg.Search.SLO, progressFn, logger))

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
				if p == nil {
					continue
				}
				result := p.result()
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
				result := p.result()
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
