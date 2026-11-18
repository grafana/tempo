package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v93/github"
)

const (
	apiPageSize                    = 100
	maxRequestAttempts             = 3
	apiRequestTimeout              = 30 * time.Second
	maxSecondaryRateLimitRetryTime = 5 * time.Minute
	maxSearchResults               = 1000
)

// newGitHubClient builds the go-github REST client used by the sweep. The HTTP
// layer wraps the default transport in retryingRoundTripper, so idempotent
// requests (GET, PUT, PATCH, DELETE) are retried up to maxRequestAttempts in
// total on transient transport errors and 5xx responses; POST requests run
// exactly once. Rate limiting and Retry-After backoff are handled by go-github
// itself, with the secondary rate-limit retry clamped to
// maxSecondaryRateLimitRetryTime.
func newGitHubClient(token string) (*github.Client, error) {
	httpClient := &http.Client{
		// No client-level Timeout: retryingRoundTripper gives each attempt its
		// own deadline so a hung attempt cannot consume the retry budget.
		Transport: &retryingRoundTripper{base: http.DefaultTransport, maxAttempts: maxRequestAttempts, backoff: retryDelay, attemptTimeout: apiRequestTimeout},
	}
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

// retryDelay returns the exponential backoff between retry attempts.
func retryDelay(attempt int) time.Duration {
	return time.Duration(attempt*2) * time.Second
}

// splitRepo splits an owner/repository string into its two parts.
func splitRepo(repo string) (owner, name string, err error) {
	owner, name, found := strings.Cut(repo, "/")
	if !found || owner == "" || name == "" {
		return "", "", fmt.Errorf("invalid repository %q; expected owner/repository", repo)
	}
	return owner, name, nil
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
