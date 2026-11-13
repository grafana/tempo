package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiVersion         = "2022-11-28"
	apiPageSize        = 100
	maxPaginationPages = 100
	maxRequestAttempts = 3
)

type apiError struct {
	method     string
	path       string
	status     int
	detail     string
	retryAfter time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s %s failed with HTTP %d: %s", e.method, e.path, e.status, strings.TrimSpace(e.detail))
}

type responseDecodeError struct {
	err error
}

func (e *responseDecodeError) Error() string { return e.err.Error() }
func (e *responseDecodeError) Unwrap() error { return e.err }

type githubClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
	// retryDelay replaces production backoff in tests without bypassing sleep cancellation.
	retryDelay func(attempt int) time.Duration
}

func newGitHubClient(token string) *githubClient {
	return &githubClient{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		retryDelay: func(attempt int) time.Duration { return time.Duration(attempt*2) * time.Second },
	}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// request executes one API operation. Safe methods retry transport errors, 429,
// and 5xx up to three times; POSTs and permanent client/decode errors run once.
// Returned headers come from the successful response and are nil on failure.
func (c *githubClient) request(
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
	out any,
) (http.Header, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode %s %s: %w", method, path, err)
		}
	}

	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		headers, err := c.requestOnce(ctx, method, path, query, payload, out)
		if err == nil {
			return headers, nil
		}
		if attempt == maxRequestAttempts || !retryable(method, err) {
			return nil, err
		}
		delay := c.retryDelay(attempt)
		var statusError *apiError
		if errors.As(err, &statusError) && statusError.retryAfter > delay {
			delay = statusError.retryAfter
		}
		if err := sleepContext(ctx, delay); err != nil {
			return nil, err
		}
	}
}

func retryable(method string, err error) bool {
	if method == http.MethodPost || errors.Is(err, context.Canceled) {
		return false
	}
	var decodeError *responseDecodeError
	if errors.As(err, &decodeError) {
		return false
	}
	var statusError *apiError
	if errors.As(err, &statusError) {
		return statusError.status == http.StatusTooManyRequests || statusError.status >= 500
	}
	return true
}

func (c *githubClient) requestOnce(
	ctx context.Context,
	method, path string,
	query url.Values,
	payload []byte,
	out any,
) (http.Header, error) {
	requestURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		requestURL = strings.TrimSuffix(c.baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	if len(query) > 0 {
		separator := "?"
		if strings.Contains(requestURL, "?") {
			separator = "&"
		}
		requestURL += separator + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("build %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "tempo-codeowner-sweep")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, readErr := io.ReadAll(io.LimitReader(resp.Body, 500))
		message := string(detail)
		if readErr != nil {
			message += fmt.Sprintf("; read error body: %v", readErr)
		}
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			message += fmt.Sprintf("; drain error body: %v", drainErr)
		}
		return nil, &apiError{
			method: method, path: path, status: resp.StatusCode,
			detail: message, retryAfter: parseRetryAfter(resp.Header, time.Now()),
		}
	}
	if out == nil {
		_, err = io.Copy(io.Discard, resp.Body)
	} else {
		err = json.NewDecoder(resp.Body).Decode(out)
	}
	if err != nil {
		return nil, &responseDecodeError{err: fmt.Errorf("decode %s %s: %w", method, path, err)}
	}
	return resp.Header, nil
}

func parseRetryAfter(headers http.Header, now time.Time) time.Duration {
	value := headers.Get("Retry-After")
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return 0
}

type pageShape uint8

const (
	bareListPage pageShape = iota
	searchItemsPage
)

// paginateList decodes GitHub's bare JSON-array list endpoints.
func paginateList[T any](ctx context.Context, client *githubClient, path string, query url.Values) ([]T, error) {
	return paginate[T](ctx, client, path, query, bareListPage)
}

// paginateSearch decodes search endpoints whose page envelope contains an items array.
func paginateSearch[T any](ctx context.Context, client *githubClient, path string, query url.Values) ([]T, error) {
	return paginate[T](ctx, client, path, query, searchItemsPage)
}

// paginate follows next links until completion. Repeated URLs and more than 100
// pages are errors rather than silent truncation or an unbounded request loop.
func paginate[T any](ctx context.Context, client *githubClient, path string, query url.Values, shape pageShape) ([]T, error) {
	results := make([]T, 0, apiPageSize)
	visited := make(map[string]struct{}, apiPageSize)
	for page := 0; path != ""; page++ {
		if page >= maxPaginationPages {
			return nil, fmt.Errorf("paginate %s exceeded %d pages", path, maxPaginationPages)
		}
		if _, ok := visited[path]; ok {
			return nil, fmt.Errorf("paginate repeated next URL %q", path)
		}
		visited[path] = struct{}{}

		var headers http.Header
		var err error
		switch shape {
		case searchItemsPage:
			var envelope struct {
				Items []T `json:"items"`
			}
			headers, err = client.request(ctx, http.MethodGet, path, query, nil, &envelope)
			results = append(results, envelope.Items...)
		case bareListPage:
			var items []T
			headers, err = client.request(ctx, http.MethodGet, path, query, nil, &items)
			results = append(results, items...)
		default:
			return nil, fmt.Errorf("paginate %s has unknown page shape %d", path, shape)
		}
		if err != nil {
			return nil, err
		}
		query = nil
		path = nextPage(headers.Get("Link"))
	}
	return results, nil
}

// nextPage returns the next relation and ignores unrelated or malformed links.
func nextPage(header string) string {
	for _, link := range strings.Split(header, ",") {
		parts := strings.Split(link, ";")
		if len(parts) < 2 {
			continue
		}
		for _, parameter := range parts[1:] {
			if strings.TrimSpace(parameter) == `rel="next"` {
				return strings.Trim(strings.TrimSpace(parts[0]), "<>")
			}
		}
	}
	return ""
}

func pageQuery() url.Values {
	return url.Values{"per_page": {strconv.Itoa(apiPageSize)}}
}

func repositoryEndpoint(repo, suffix string) string {
	return "repos/" + repo + suffix
}

func issueEndpoint(repo string, number int, suffix string) string {
	return fmt.Sprintf("repos/%s/issues/%d%s", repo, number, suffix)
}
