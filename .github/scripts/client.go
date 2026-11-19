package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v93/github"
)

// newGitHubClient builds the plain go-github REST client used by the sweep.
// The client timeout bounds each request so a hung response cannot stall the
// run; go-github handles rate-limit and Retry-After backoff internally, with
// the secondary rate-limit retry clamped to maxSecondaryRateLimitRetryTime. We
// layer no retries on top — if transient failures become a problem, add them
// back at the caller.
func newGitHubClient(token string) (*github.Client, error) {
	httpClient := &http.Client{Timeout: apiRequestTimeout}
	client, err := github.NewClient(
		github.WithHTTPClient(httpClient),
		github.WithAuthToken(token),
		github.WithMaxSecondaryRateLimitRetryAfterDuration(maxSecondaryRateLimitRetryTime),
	)
	if err != nil {
		return nil, fmt.Errorf("create GitHub client: %w", err)
	}
	return client, nil
}

// eachPage collects every page of a paginated go-github list call. The next
// closure must copy the supplied ListOptions into the service's own options.
func eachPage[T any](ctx context.Context, next func(*github.ListOptions) ([]T, *github.Response, error)) ([]T, error) {
	return eachPageBounded(ctx, next, 0)
}

// eachPageBounded is eachPage with a cap on collected items. maxItems <= 0
// means unbounded. Search endpoints return at most maxSearchResults regardless
// of pagination, and requesting page 11 of a >1000-result search is a 422, so
// the closed-load search must stop at the cap instead of following the link
// and aborting the sweep.
func eachPageBounded[T any](ctx context.Context, next func(*github.ListOptions) ([]T, *github.Response, error), maxItems int) ([]T, error) {
	var all []T
	opts := &github.ListOptions{PerPage: apiPageSize}
	for {
		items, resp, err := next(opts)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if maxItems > 0 && len(all) >= maxItems {
			return all, nil
		}
		if resp.NextPage == 0 {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
}
