package datadog

import (
	"fmt"
	"net/http"
	"net/url"
)

// The list of Rate Limited Endpoints of the Datadog API.
// https://docs.datadoghq.com/api/?lang=bash#rate-limiting
func (client *Client) updateRateLimits(resp *http.Response, api *url.URL) error {
	if resp == nil || resp.Header == nil || api.Path == "" {
		return fmt.Errorf("malformed HTTP content.")
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		// The endpoint is not Rate Limited.
		return nil
	}
	client.m.Lock()
	defer client.m.Unlock()
	client.rateLimitingStats[api.Path] = RateLimit{
		Limit:     resp.Header.Get("X-RateLimit-Limit"),
		Reset:     resp.Header.Get("X-RateLimit-Reset"),
		Period:    resp.Header.Get("X-RateLimit-Period"),
		Remaining: resp.Header.Get("X-RateLimit-Remaining"),
	}
	return nil
}

// GetRateLimitStats is a threadsafe getter to retrieve the rate limiting stats associated with the Client.
func (client *Client) GetRateLimitStats() map[string]RateLimit {
	client.m.Lock()
	defer client.m.Unlock()
	// Shallow copy to avoid corrupted data
	mapCopy := make(map[string]RateLimit, len(client.rateLimitingStats))
	for k, v := range client.rateLimitingStats {
		mapCopy[k] = v
	}
	return mapCopy
}
