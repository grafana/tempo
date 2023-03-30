// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package writer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/telemetry"
)

// newSenders returns a list of senders based on the given agent configuration, using climit
// as the maximum number of concurrent outgoing connections, writing to path.
func newSenders(cfg *config.AgentConfig, r eventRecorder, path string, climit, qsize int, telemetryCollector telemetry.TelemetryCollector) []*sender {
	if e := cfg.Endpoints; len(e) == 0 || e[0].Host == "" || e[0].APIKey == "" {
		panic(errors.New("config was not properly validated"))
	}
	// spread out the the maximum connection limit (climit) between senders
	maxConns := math.Max(1, float64(climit/len(cfg.Endpoints)))
	senders := make([]*sender, len(cfg.Endpoints))
	for i, endpoint := range cfg.Endpoints {
		url, err := url.Parse(endpoint.Host + path)
		if err != nil {
			telemetryCollector.SendStartupError(telemetry.InvalidIntakeEndpoint, err)
			log.Criticalf("Invalid host endpoint: %q", endpoint.Host)
			os.Exit(1)
		}
		senders[i] = newSender(&senderConfig{
			client:    cfg.NewHTTPClient(),
			maxConns:  int(maxConns),
			maxQueued: qsize,
			url:       url,
			apiKey:    endpoint.APIKey,
			recorder:  r,
			userAgent: fmt.Sprintf("Datadog Trace Agent/%s/%s", cfg.AgentVersion, cfg.GitCommit),
		})
	}
	return senders
}

// eventRecorder implementations are able to take note of events happening in
// the sender.
type eventRecorder interface {
	// recordEvent notifies that event t has happened, passing details about
	// the event in data.
	recordEvent(t eventType, data *eventData)
}

// eventType specifies an event which occurred in the sender.
type eventType int

const (
	// eventTypeRetry specifies that a send failed with a retriable error (5xx).
	eventTypeRetry eventType = iota
	// eventTypeSent specifies that a single payload was successfully sent.
	eventTypeSent
	// eventTypeRejected specifies that the edge rejected this payload.
	eventTypeRejected
	// eventTypeDropped specifies that a payload had to be dropped to make room
	// in the queue.
	eventTypeDropped
)

var eventTypeStrings = map[eventType]string{
	eventTypeRetry:    "eventTypeRetry",
	eventTypeSent:     "eventTypeSent",
	eventTypeRejected: "eventTypeRejected",
	eventTypeDropped:  "eventTypeDropped",
}

// String implements fmt.Stringer.
func (t eventType) String() string { return eventTypeStrings[t] }

// eventData represents information about a sender event. Not all fields apply
// to all events.
type eventData struct {
	// host specifies the host which the sender is sending to.
	host string
	// bytes represents the number of bytes affected by this event.
	bytes int
	// count specfies the number of payloads that this events refers to.
	count int
	// duration specifies the time it took to complete this event. It
	// is set for eventType{Sent,Retry,Rejected}.
	duration time.Duration
	// err specifies the error that may have occurred on events eventType{Retry,Rejected}.
	err error
	// connectionFill specifies the percentage of allowed connections used.
	// At 100% (1.0) the writer will become blocking.
	connectionFill float64
	// queueFill specifies how flul the queue is. It's a floating point number ranging
	// between 0 (0%) and 1 (100%).
	queueFill float64
}

// senderConfig specifies the configuration for the sender.
type senderConfig struct {
	// client specifies the HTTP client to use when sending requests.
	client *config.ResetClient
	// url specifies the URL to send requests too.
	url *url.URL
	// apiKey specifies the Datadog API key to use.
	apiKey string
	// maxConns specifies the maximum number of allowed concurrent ougoing
	// connections.
	maxConns int
	// maxQueued specifies the maximum number of payloads allowed in the queue.
	// When it is surpassed, oldest items get dropped to make room for new ones.
	maxQueued int
	// recorder specifies the eventRecorder to use when reporting events occurring
	// in the sender.
	recorder eventRecorder
	// userAgent is the computed user agent we'll use when communicating with Datadog
	userAgent string
}

// sender is responsible for sending payloads to a given URL. It uses a size-limited
// retry queue with a backoff mechanism in case of retriable errors.
type sender struct {
	cfg *senderConfig

	queue    chan *payload // payload queue
	climit   chan struct{} // semaphore for limiting concurrent connections
	inflight *atomic.Int32 // inflight payloads
	attempt  *atomic.Int32 // active retry attempt

	mu     sync.RWMutex // guards closed
	closed bool         // closed reports if the loop is stopped
}

// newSender returns a new sender based on the given config cfg.
func newSender(cfg *senderConfig) *sender {
	s := sender{
		cfg:      cfg,
		queue:    make(chan *payload, cfg.maxQueued),
		climit:   make(chan struct{}, cfg.maxConns),
		inflight: atomic.NewInt32(0),
		attempt:  atomic.NewInt32(0),
	}
	go s.loop()
	return &s
}

// loop runs the main sender loop.
func (s *sender) loop() {
	for p := range s.queue {
		s.backoff()
		s.climit <- struct{}{}
		go func(p *payload) {
			defer func() { <-s.climit }()
			s.sendPayload(p)
		}(p)
	}
}

// backoff triggers a sleep period proportional to the retry attempt, if any.
func (s *sender) backoff() {
	attempt := s.attempt.Load()
	delay := backoffDuration(int(attempt))
	if delay == 0 {
		return
	}
	time.Sleep(delay)
}

// Stop stops the sender. It attempts to wait for all inflight payloads to complete
// with a timeout of 5 seconds.
func (s *sender) Stop() {
	s.WaitForInflight()
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	close(s.queue)
}

// WaitForInflight blocks until all in progress payloads are sent,
// or the timeout is reached.
func (s *sender) WaitForInflight() {
	timeout := time.After(5 * time.Second)
outer:
	for {
		select {
		case <-timeout:
			break outer
		default:
			if s.inflight.Load() == 0 {
				break outer
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Push pushes p onto the sender's queue, to be written to the destination.
func (s *sender) Push(p *payload) {
	for {
		select {
		case s.queue <- p:
			// ok
			s.inflight.Inc()
			return
		default:
			// drop the oldest item in the queue to make room
			select {
			case p := <-s.queue:
				s.releasePayload(p, eventTypeDropped, &eventData{
					bytes: p.body.Len(),
					count: 1,
				})
			default:
				// the queue got drained; not very likely to happen, but
				// we shouldn't risk a deadlock
				continue
			}
		}
	}
}

// sendPayload sends the payload p to the destination URL.
func (s *sender) sendPayload(p *payload) {
	req, err := p.httpRequest(s.cfg.url)
	if err != nil {
		log.Errorf("http.Request: %s", err)
		return
	}
	start := time.Now()
	err = s.do(req)
	stats := &eventData{
		bytes:    p.body.Len(),
		count:    1,
		duration: time.Since(start),
		err:      err,
	}
	switch err.(type) {
	case *retriableError:
		// request failed again, but can be retried
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.closed {
			// sender is stopped
			return
		}
		s.attempt.Inc()

		if r := p.retries.Inc(); (r&(r-1)) == 0 && r > 3 {
			// Only log a warning if the retry attempt is a power of 2
			// and larger than 3, to avoid alerting the user unnecessarily.
			// e.g. attempts 4, 8, 16, etc.
			log.Warnf("Retried payload %d times: %s", r, err.Error())
		}
		select {
		case s.queue <- p:
			s.recordEvent(eventTypeRetry, stats)
			return
		default:
			// queue is full; since this is the oldest payload, we drop it
			s.releasePayload(p, eventTypeDropped, stats)
		}
	case nil:
		// request was successful; the retry queue may have grown large - we should
		// reduce the backoff gradually to avoid hitting the edge too hard.
		for {
			// interlock with other sends to avoid setting the same value
			attempt := s.attempt.Load()
			if s.attempt.CAS(attempt, attempt/2) {
				break
			}
		}
		s.releasePayload(p, eventTypeSent, stats)
	default:
		// this is a fatal error, we have to drop this payload
		s.releasePayload(p, eventTypeRejected, stats)
	}
}

// waitForSenders blocks until all senders have sent their inflight payloads
func waitForSenders(senders []*sender) {
	var wg sync.WaitGroup
	for _, s := range senders {
		wg.Add(1)
		go func(s *sender) {
			defer wg.Done()
			s.WaitForInflight()
		}(s)
	}
	wg.Wait()
}

// releasePayload releases the payload p and records the specified event. The payload
// should not be used again after a release.
func (s *sender) releasePayload(p *payload, t eventType, data *eventData) {
	s.recordEvent(t, data)
	ppool.Put(p)
	s.inflight.Dec()
}

// recordEvent records the occurrence of the given event type t. It additionally
// passes on the data and augments it with additional information.
func (s *sender) recordEvent(t eventType, data *eventData) {
	if s.cfg.recorder == nil {
		return
	}
	data.host = s.cfg.url.Hostname()
	data.connectionFill = float64(len(s.climit)) / float64(cap(s.climit))
	data.queueFill = float64(len(s.queue)) / float64(cap(s.queue))
	s.cfg.recorder.recordEvent(t, data)
}

// retriableError is an error returned by the server which may be retried at a later time.
type retriableError struct{ err error }

// Error implements error.
func (e retriableError) Error() string { return e.err.Error() }

const (
	headerAPIKey    = "DD-Api-Key"
	headerUserAgent = "User-Agent"
)

func (s *sender) do(req *http.Request) error {
	req.Header.Set(headerAPIKey, s.cfg.apiKey)
	req.Header.Set(headerUserAgent, s.cfg.userAgent)
	resp, err := s.cfg.client.Do(req)
	if err != nil {
		// request errors include timeouts or name resolution errors and
		// should thus be retried.
		return &retriableError{err}
	}
	// From https://golang.org/pkg/net/http/#Response:
	// The default HTTP client's Transport may not reuse HTTP/1.x "keep-alive"
	// TCP connections if the Body is not read to completion and closed.
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		log.Debugf("Error discarding request body: %v", err)
	}
	resp.Body.Close()

	if isRetriable(resp.StatusCode) {
		return &retriableError{
			fmt.Errorf("server responded with %q", resp.Status),
		}
	}
	if resp.StatusCode/100 != 2 {
		// status codes that are neither 2xx nor 5xx are considered
		// non-retriable failures
		return errors.New(resp.Status)
	}
	return nil
}

// isRetriable reports whether the give HTTP status code should be retried.
func isRetriable(code int) bool {
	if code == http.StatusRequestTimeout {
		return true
	}
	// 5xx errors can be retried
	return code/100 == 5
}

// payloads specifies a payload to be sent by the sender.
type payload struct {
	body    *bytes.Buffer     // request body
	headers map[string]string // request headers
	retries *atomic.Int32     // number of retries sending this payload
}

// ppool is a pool of payloads.
var ppool = &sync.Pool{
	New: func() interface{} {
		return &payload{
			body:    &bytes.Buffer{},
			headers: make(map[string]string),
			retries: atomic.NewInt32(0),
		}
	},
}

// newPayload returns a new payload with the given headers. The payload should not
// be used anymore after it has been given to the sender.
func newPayload(headers map[string]string) *payload {
	p := ppool.Get().(*payload)
	p.body.Reset()
	p.headers = headers
	p.retries.Store(0)
	return p
}

func (p *payload) clone() *payload {
	headers := make(map[string]string, len(p.headers))
	for k, v := range p.headers {
		headers[k] = v
	}
	clone := newPayload(headers)
	if _, err := clone.body.ReadFrom(bytes.NewBuffer(p.body.Bytes())); err != nil {
		log.Errorf("Error cloning writer payload: %v", err)
	}
	return clone
}

// httpRequest returns an HTTP request based on the payload, targeting the given URL.
func (p *payload) httpRequest(url *url.URL) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(p.body.Bytes()))
	if err != nil {
		// this should never happen with sanitized data (invalid method or invalid url)
		return nil, err
	}
	for k, v := range p.headers {
		req.Header.Add(k, v)
	}
	req.Header.Add("Content-Length", strconv.Itoa(p.body.Len()))
	return req, nil
}

// stopSenders attempts to simultaneously stop a group of senders.
func stopSenders(senders []*sender) {
	var wg sync.WaitGroup
	for _, s := range senders {
		wg.Add(1)
		go func(s *sender) {
			defer wg.Done()
			s.Stop()
		}(s)
	}
	wg.Wait()
}

// sendPayloads sends the payload p to all senders.
func sendPayloads(senders []*sender, p *payload, syncMode bool) {
	if syncMode {
		defer waitForSenders(senders)
	}

	if len(senders) == 1 {
		// fast path
		senders[0].Push(p)
		return
	}
	// Create a clone for each payload because each sender places payloads
	// back onto the pool after they are sent.
	payloads := make([]*payload, 0, len(senders))
	// Perform all the clones before any sends are to ensure the original
	// payload body is completely unread.
	for i := range senders {
		if i == 0 {
			payloads = append(payloads, p)
		} else {
			payloads = append(payloads, p.clone())
		}
	}
	for i, sender := range senders {
		sender.Push(payloads[i])
	}
}

const (
	// backoffBase specifies the multiplier base for the backoff duration algorithm.
	backoffBase = 100 * time.Millisecond
	// backoffMaxDuration is the maximum permitted backoff duration.
	backoffMaxDuration = 10 * time.Second
)

// backoffDuration returns the backoff duration necessary for the given attempt.
// The formula is "Full Jitter":
//
//	random_between(0, min(cap, base * 2 ** attempt))
//
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
var backoffDuration = func(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}
	maxPow := float64(backoffMaxDuration / backoffBase)
	pow := math.Min(math.Pow(2, float64(attempt)), maxPow)
	ns := int64(float64(backoffBase) * pow)
	return time.Duration(rand.Int63n(ns))
}
