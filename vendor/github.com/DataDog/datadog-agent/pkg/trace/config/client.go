// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"net/http"
	"sync"
	"time"
)

// TODO(gbbr): Perhaps this is not the best place for this structure.

// ResetClient wraps (http.Client).Do and resets the underlying connections at the
// configured interval
type ResetClient struct {
	httpClientFactory func() *http.Client
	resetInterval     time.Duration

	mu         sync.RWMutex
	httpClient *http.Client
	lastReset  time.Time
}

// NewResetClient returns an initialized Client resetting connections at the passed resetInterval ("0"
// means that no reset is performed).
// The underlying http.Client used will be created using the passed http client factory.
func NewResetClient(resetInterval time.Duration, httpClientFactory func() *http.Client) *ResetClient {
	return &ResetClient{
		httpClientFactory: httpClientFactory,
		resetInterval:     resetInterval,
		httpClient:        httpClientFactory(),
		lastReset:         time.Now(),
	}
}

// Do wraps (http.Client).Do. Thread safe.
func (c *ResetClient) Do(req *http.Request) (*http.Response, error) {
	c.checkReset()

	c.mu.RLock()
	httpClient := c.httpClient
	c.mu.RUnlock()

	return httpClient.Do(req)
}

// checkReset checks whether a client reset should be performed, and performs it
// if so
func (c *ResetClient) checkReset() {
	if c.resetInterval == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.lastReset) < c.resetInterval {
		return
	}

	c.lastReset = time.Now()
	// Close idle connections on underlying client. Safe to do while other goroutines use the client.
	// This is a best effort: if other goroutine(s) are currently using the client,
	// the related open connection(s) will remain open until the client is GC'ed
	c.httpClient.CloseIdleConnections()
	c.httpClient = c.httpClientFactory()
}
