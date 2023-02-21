// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package writer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
)

// uid is an atomically incremented ID, used by the expectResponses function to
// create payload IDs for the test server.
var uid = atomic.NewUint64(0)

// expectResponses creates a new payload for the test server. The test server will
// respond with the given status codes, in the given order, for each subsequent
// request, in rotation.
func expectResponses(codes ...int) *payload {
	if len(codes) == 0 {
		codes = []int{http.StatusOK}
	}
	p := newPayload(nil)
	p.body.WriteString(strconv.FormatUint(uid.Inc(), 10))
	p.body.WriteString("|")
	for i, code := range codes {
		if i > 0 {
			p.body.WriteString(",")
		}
		p.body.WriteString(strconv.Itoa(code))
	}
	return p
}

// newTestServerWithLatency returns a test server that takes duration d
// to respond to each request.
func newTestServerWithLatency(d time.Duration) *testServer {
	ts := newTestServer()
	ts.latency = d
	return ts
}

// newTestServer returns a new, started HTTP test server. Its URL is available
// as a field. To control its responses, send it payloads created by expectResponses.
// By default, the testServer always returns http.StatusOK.
func newTestServer() *testServer {
	srv := &testServer{
		seen:     make(map[string]*requestStatus),
		total:    atomic.NewUint64(0),
		accepted: atomic.NewUint64(0),
		retried:  atomic.NewUint64(0),
		failed:   atomic.NewUint64(0),
		peak:     atomic.NewInt64(0),
		active:   atomic.NewInt64(0),
	}
	srv.server = httptest.NewServer(srv)
	srv.URL = srv.server.URL
	return srv
}

// testServer is an http.Handler and http.Server which records the number of total,
// failed, retriable and accepted requests. It also allows manipulating it's HTTTP
// status code response by means of the request's body (see expectResponses).
type testServer struct {
	URL     string
	server  *httptest.Server
	latency time.Duration

	mu       sync.Mutex // guards below
	seen     map[string]*requestStatus
	payloads []*payload

	// stats
	total, accepted *atomic.Uint64
	retried, failed *atomic.Uint64
	peak, active    *atomic.Int64
}

// requestStatus keeps track of how many times a custom payload was seen and what
// the next HTTP status code response should be.
type requestStatus struct {
	count int
	codes []int
}

// nextResponse returns the next HTTP response code and advances the count.
func (rs *requestStatus) nextResponse() int {
	statusCode := rs.codes[rs.count%len(rs.codes)]
	rs.count++
	return statusCode
}

// Peak returns the maximum number of simultaneous connections that were active
// while the server was running.
func (ts *testServer) Peak() int { return int(ts.peak.Load()) }

// Failed returns the number of connections to which the server responded with an
// HTTP status code that is non-2xx and non-5xx.
func (ts *testServer) Failed() int { return int(ts.failed.Load()) }

// Failed returns the number of connections to which the server responded with a
// 5xx HTTP status code.
func (ts *testServer) Retried() int { return int(ts.retried.Load()) }

// Total returns the total number of connections which reached the server.
func (ts *testServer) Total() int { return int(ts.total.Load()) }

// Failed returns the number of connections to which the server responded with a
// 2xx HTTP status code.
func (ts *testServer) Accepted() int { return int(ts.accepted.Load()) }

// Payloads returns the payloads that were accepted by the server, as received.
func (ts *testServer) Payloads() []*payload {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.payloads
}

// ServeHTTP responds based on the request body.
func (ts *testServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ts.total.Inc()

	if v := ts.active.Inc(); v > ts.peak.Load() {
		ts.peak.Swap(v)
	}
	defer ts.active.Dec()
	if ts.latency > 0 {
		time.Sleep(ts.latency)
	}

	slurp, err := io.ReadAll(req.Body)
	if err != nil {
		panic(fmt.Sprintf("error reading request body: %v", err))
	}
	defer req.Body.Close()
	statusCode := ts.getNextCode(slurp)
	w.WriteHeader(statusCode)
	switch {
	case isRetriable(statusCode):
		ts.retried.Inc()
	case statusCode/100 == 2: // 2xx
		ts.accepted.Inc()
		// for 2xx, we store the payload contents too
		headers := make(map[string]string, len(req.Header))
		for k, vs := range req.Header {
			for _, v := range vs {
				headers[k] = v
			}
		}
		ts.mu.Lock()
		defer ts.mu.Unlock()
		ts.payloads = append(ts.payloads, &payload{
			body:    bytes.NewBuffer(slurp),
			headers: headers,
		})
	default:
		ts.failed.Inc()
	}
}

// getNextCode returns the next HTTP status code that should be responded with
// to the given request body. If the request body does not originate from a
// payload created with expectResponse, it returns http.StatusOK.
func (ts *testServer) getNextCode(reqBody []byte) int {
	parts := strings.Split(string(reqBody), "|")
	if len(parts) != 2 {
		// not a special body
		return http.StatusOK
	}
	id := parts[0]
	ts.mu.Lock()
	defer ts.mu.Unlock()
	p, ok := ts.seen[id]
	if !ok {
		parts := strings.Split(parts[1], ",")
		codes := make([]int, len(parts))
		for i, part := range parts {
			code, err := strconv.Atoi(part)
			if err != nil {
				// this is likely a real proto request or something else; never the less, let's
				// ensure the user knows, just in case it wasn't meant to be.
				log.Println("testServer: warning: possibly malformed request body")
				return http.StatusOK
			}
			if http.StatusText(code) == "" {
				panic(fmt.Sprintf("testServer: invalid status code: %d", code))
			}
			codes[i] = code
		}
		ts.seen[id] = &requestStatus{codes: codes}
		p = ts.seen[id]
	}
	return p.nextResponse()
}

// Close closes the underlying http.Server.
func (ts *testServer) Close() { ts.server.Close() }
