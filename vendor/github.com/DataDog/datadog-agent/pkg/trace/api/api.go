// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"math"
	"mime"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinylib/msgp/msgp"
	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/api/apiutil"
	"github.com/DataDog/datadog-agent/pkg/trace/api/internal/header"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics/timing"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/trace/telemetry"
	"github.com/DataDog/datadog-agent/pkg/trace/watchdog"
)

// outOfCPULogThreshold is used to throttle the out-of-cpu warnning logs
// i.e we log the warning on every outOfCPULogThreshold occurrences.
// The value 10 is based on load test experiments and can be revisited in the future.
const outOfCPULogThreshold uint32 = 10

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	return buffer
}

func putBuffer(buffer *bytes.Buffer) {
	bufferPool.Put(buffer)
}

// HTTPReceiver is a collector that uses HTTP protocol and just holds
// a chan where the spans received are sent one by one
type HTTPReceiver struct {
	Stats       *info.ReceiverStats
	RateLimiter *rateLimiter

	out                 chan *Payload
	conf                *config.AgentConfig
	dynConf             *sampler.DynamicConfig
	server              *http.Server
	statsProcessor      StatsProcessor
	containerIDProvider IDProvider

	telemetryCollector telemetry.TelemetryCollector

	rateLimiterResponse int // HTTP status code when refusing

	wg   sync.WaitGroup // waits for all requests to be processed
	exit chan struct{}

	// outOfCPUCounter is counter to throttle the out of cpu warning log
	outOfCPUCounter *atomic.Uint32
}

// NewHTTPReceiver returns a pointer to a new HTTPReceiver
func NewHTTPReceiver(conf *config.AgentConfig, dynConf *sampler.DynamicConfig, out chan *Payload, statsProcessor StatsProcessor, telemetryCollector telemetry.TelemetryCollector) *HTTPReceiver {
	rateLimiterResponse := http.StatusOK
	if conf.HasFeature("429") {
		rateLimiterResponse = http.StatusTooManyRequests
	}
	return &HTTPReceiver{
		Stats:       info.NewReceiverStats(),
		RateLimiter: newRateLimiter(),

		out:                 out,
		statsProcessor:      statsProcessor,
		conf:                conf,
		dynConf:             dynConf,
		containerIDProvider: NewIDProvider(conf.ContainerProcRoot),

		telemetryCollector: telemetryCollector,

		rateLimiterResponse: rateLimiterResponse,

		exit: make(chan struct{}),

		outOfCPUCounter: atomic.NewUint32(0),
	}
}

func (r *HTTPReceiver) buildMux() *http.ServeMux {
	mux := http.NewServeMux()

	hash, infoHandler := r.makeInfoHandler()
	for _, e := range endpoints {
		if e.IsEnabled != nil && !e.IsEnabled(r.conf) {
			continue
		}
		mux.Handle(e.Pattern, replyWithVersion(hash, r.conf.AgentVersion, e.Handler(r)))
	}
	mux.HandleFunc("/info", infoHandler)

	return mux
}

// replyWithVersion returns an http.Handler which calls h with an addition of some
// HTTP headers containing version and state information.
func replyWithVersion(hash string, version string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Datadog-Agent-Version", version)
		w.Header().Set("Datadog-Agent-State", hash)
		h.ServeHTTP(w, r)
	})
}

// Start starts doing the HTTP server and is ready to receive traces
func (r *HTTPReceiver) Start() {
	if r.conf.ReceiverPort == 0 {
		log.Debug("HTTP receiver disabled by config (apm_config.receiver_port: 0).")
		return
	}

	timeout := 5 * time.Second
	if r.conf.ReceiverTimeout > 0 {
		timeout = time.Duration(r.conf.ReceiverTimeout) * time.Second
	}
	httpLogger := log.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	r.server = &http.Server{
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		ErrorLog:     stdlog.New(httpLogger, "http.Server: ", 0),
		Handler:      r.buildMux(),
		ConnContext:  connContext,
	}

	addr := net.JoinHostPort(r.conf.ReceiverHost, strconv.Itoa(r.conf.ReceiverPort))
	ln, err := r.listenTCP(addr)
	if err != nil {
		r.telemetryCollector.SendStartupError(telemetry.CantStartHttpServer, err)
		killProcess("Error creating tcp listener: %v", err)
	}
	go func() {
		defer watchdog.LogOnPanic()
		if err := r.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Errorf("Could not start HTTP server: %v. HTTP receiver disabled.", err)
			r.telemetryCollector.SendStartupError(telemetry.CantStartHttpServer, err)
		}
	}()
	log.Infof("Listening for traces at http://%s", addr)

	if path := r.conf.ReceiverSocket; path != "" {
		ln, err := r.listenUnix(path)
		if err != nil {
			r.telemetryCollector.SendStartupError(telemetry.CantStartUdsServer, err)
			killProcess("Error creating UDS listener: %v", err)
		}
		go func() {
			defer watchdog.LogOnPanic()
			if err := r.server.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Errorf("Could not start UDS server: %v. UDS receiver disabled.", err)
				r.telemetryCollector.SendStartupError(telemetry.CantStartUdsServer, err)
			}
		}()
		log.Infof("Listening for traces at unix://%s", path)
	}

	if path := r.conf.WindowsPipeName; path != "" {
		pipepath := `\\.\pipe\` + path
		bufferSize := r.conf.PipeBufferSize
		secdec := r.conf.PipeSecurityDescriptor
		ln, err := listenPipe(pipepath, secdec, bufferSize)
		if err != nil {
			r.telemetryCollector.SendStartupError(telemetry.CantStartWindowsPipeServer, err)
			killProcess("Error creating %q named pipe: %v", pipepath, err)
		}
		go func() {
			defer watchdog.LogOnPanic()
			if err := r.server.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Errorf("Could not start Windows Pipes server: %v. Windows Pipes receiver disabled.", err)
				r.telemetryCollector.SendStartupError(telemetry.CantStartWindowsPipeServer, err)
			}
		}()
		log.Infof("Listening for traces on Windowes pipe %q. Security descriptor is %q", pipepath, secdec)
	}

	go r.RateLimiter.Run()

	go func() {
		defer watchdog.LogOnPanic()
		r.loop()
	}()
}

// listenUnix returns a net.Listener listening on the given "unix" socket path.
func (r *HTTPReceiver) listenUnix(path string) (net.Listener, error) {
	fi, err := os.Stat(path)
	if err == nil {
		// already exists
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("cannot reuse %q; not a unix socket", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("unable to remove stale socket: %v", err)
		}
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o722); err != nil {
		return nil, fmt.Errorf("error setting socket permissions: %v", err)
	}
	return NewMeasuredListener(ln, "uds_connections"), err
}

// listenTCP creates a new net.Listener on the provided TCP address.
func (r *HTTPReceiver) listenTCP(addr string) (net.Listener, error) {
	tcpln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if climit := r.conf.ConnectionLimit; climit > 0 {
		ln, err := newRateLimitedListener(tcpln, climit)
		go func() {
			defer watchdog.LogOnPanic()
			ln.Refresh(climit)
		}()
		return ln, err
	}
	return NewMeasuredListener(tcpln, "tcp_connections"), err
}

// Stop stops the receiver and shuts down the HTTP server.
func (r *HTTPReceiver) Stop() error {
	if r.conf.ReceiverPort == 0 {
		return nil
	}
	r.exit <- struct{}{}
	<-r.exit

	r.RateLimiter.Stop()

	expiry := time.Now().Add(5 * time.Second) // give it 5 seconds
	ctx, cancel := context.WithDeadline(context.Background(), expiry)
	defer cancel()
	if err := r.server.Shutdown(ctx); err != nil {
		return err
	}
	r.wg.Wait()
	close(r.out)
	return nil
}

func (r *HTTPReceiver) handleWithVersion(v Version, f func(Version, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if mediaType := getMediaType(req); mediaType == "application/msgpack" && (v == v01 || v == v02) {
			// msgpack is only supported for versions >= v0.3
			httpFormatError(w, v, fmt.Errorf("unsupported media type: %q", mediaType))
			return
		}

		if req.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "cross-site request rejected", http.StatusForbidden)
			return
		}

		// TODO(x): replace with http.MaxBytesReader?
		req.Body = apiutil.NewLimitedReader(req.Body, r.conf.MaxRequestBytes)

		f(v, w, req)
	}
}

var errInvalidHeaderTraceCountValue = fmt.Errorf("%q header value is not a number", header.TraceCount)

func traceCount(req *http.Request) (int64, error) {
	str := req.Header.Get(header.TraceCount)
	if str == "" {
		return 0, fmt.Errorf("HTTP header %q not found", header.TraceCount)
	}
	n, err := strconv.Atoi(str)
	if err != nil {
		return 0, errInvalidHeaderTraceCountValue
	}
	return int64(n), nil
}

const (
	// tagContainersTags specifies the name of the tag which holds key/value
	// pairs representing information about the container (Docker, EC2, etc).
	tagContainersTags = "_dd.tags.container"
)

// TagStats returns the stats and tags coinciding with the information found in header.
// For more information, check the "Datadog-Meta-*" HTTP headers defined in this file.
func (r *HTTPReceiver) TagStats(v Version, header http.Header) *info.TagStats {
	return r.tagStats(v, header)
}

func (r *HTTPReceiver) tagStats(v Version, httpHeader http.Header) *info.TagStats {
	return r.Stats.GetTagStats(info.Tags{
		Lang:            httpHeader.Get(header.Lang),
		LangVersion:     httpHeader.Get(header.LangVersion),
		Interpreter:     httpHeader.Get(header.LangInterpreter),
		LangVendor:      httpHeader.Get(header.LangInterpreterVendor),
		TracerVersion:   httpHeader.Get(header.TracerVersion),
		EndpointVersion: string(v),
	})
}

// decodeTracerPayload decodes the payload in http request `req`.
// - tp is the decoded payload
// - ranHook reports whether the decoder was able to run the pb.MetaHook
// - err is the first error encountered
func decodeTracerPayload(v Version, req *http.Request, ts *info.TagStats, cIDProvider IDProvider) (tp *pb.TracerPayload, ranHook bool, err error) {
	switch v {
	case v01:
		var spans []pb.Span
		if err = json.NewDecoder(req.Body).Decode(&spans); err != nil {
			return nil, false, err
		}
		return &pb.TracerPayload{
			LanguageName:    ts.Lang,
			LanguageVersion: ts.LangVersion,
			ContainerID:     cIDProvider.GetContainerID(req.Context(), req.Header),
			Chunks:          traceChunksFromSpans(spans),
			TracerVersion:   ts.TracerVersion,
		}, false, nil
	case v05:
		buf := getBuffer()
		defer putBuffer(buf)
		if _, err = io.Copy(buf, req.Body); err != nil {
			return nil, false, err
		}
		var traces pb.Traces
		err = traces.UnmarshalMsgDictionary(buf.Bytes())
		return &pb.TracerPayload{
			LanguageName:    ts.Lang,
			LanguageVersion: ts.LangVersion,
			ContainerID:     cIDProvider.GetContainerID(req.Context(), req.Header),
			Chunks:          traceChunksFromTraces(traces),
			TracerVersion:   ts.TracerVersion,
		}, true, err
	case V07:
		buf := getBuffer()
		defer putBuffer(buf)
		if _, err = io.Copy(buf, req.Body); err != nil {
			return nil, false, err
		}
		var tracerPayload pb.TracerPayload
		_, err = tracerPayload.UnmarshalMsg(buf.Bytes())
		return &tracerPayload, true, err
	default:
		var traces pb.Traces
		if ranHook, err = decodeRequest(req, &traces); err != nil {
			return nil, false, err
		}
		return &pb.TracerPayload{
			LanguageName:    ts.Lang,
			LanguageVersion: ts.LangVersion,
			ContainerID:     cIDProvider.GetContainerID(req.Context(), req.Header),
			Chunks:          traceChunksFromTraces(traces),
			TracerVersion:   ts.TracerVersion,
		}, ranHook, nil
	}
}

// replyOK replies to the given http.ResponseWriter w based on the endpoint version, with either status 200/OK
// or with a list of rates by service. It returns the number of bytes written along with reporting if the operation
// was successful.
func (r *HTTPReceiver) replyOK(req *http.Request, v Version, w http.ResponseWriter) (n uint64, ok bool) {
	switch v {
	case v01, v02, v03:
		return httpOK(w)
	default:
		ratesVersion := req.Header.Get(header.RatesPayloadVersion)
		return httpRateByService(ratesVersion, w, r.dynConf)
	}
}

// rateLimited reports whether n number of traces should be rejected by the API.
func (r *HTTPReceiver) rateLimited(n int64) bool {
	if n == 0 {
		return false
	}
	if r.conf.MaxMemory == 0 && r.conf.MaxCPU == 0 {
		// rate limiting is off
		return false
	}
	return !r.RateLimiter.Permits(n)
}

// StatsProcessor implementations are able to process incoming client stats.
type StatsProcessor interface {
	// ProcessStats takes a stats payload and consumes it. It is considered to be originating
	// from the given lang.
	ProcessStats(p pb.ClientStatsPayload, lang, tracerVersion string)
}

// handleStats handles incoming stats payloads.
func (r *HTTPReceiver) handleStats(w http.ResponseWriter, req *http.Request) {
	defer timing.Since("datadog.trace_agent.receiver.stats_process_ms", time.Now())

	ts := r.tagStats(V07, req.Header)
	rd := apiutil.NewLimitedReader(req.Body, r.conf.MaxRequestBytes)
	req.Header.Set("Accept", "application/msgpack")
	var in pb.ClientStatsPayload
	if err := msgp.Decode(rd, &in); err != nil {
		log.Errorf("Error decoding pb.ClientStatsPayload: %v", err)
		httpDecodingError(err, []string{"handler:stats", "codec:msgpack", "v:v0.6"}, w)
		return
	}

	metrics.Count("datadog.trace_agent.receiver.stats_payload", 1, ts.AsTags(), 1)
	metrics.Count("datadog.trace_agent.receiver.stats_bytes", rd.Count, ts.AsTags(), 1)
	metrics.Count("datadog.trace_agent.receiver.stats_buckets", int64(len(in.Stats)), ts.AsTags(), 1)

	r.statsProcessor.ProcessStats(in, req.Header.Get(header.Lang), req.Header.Get(header.TracerVersion))
}

// handleTraces knows how to handle a bunch of traces
func (r *HTTPReceiver) handleTraces(v Version, w http.ResponseWriter, req *http.Request) {
	ts := r.tagStats(v, req.Header)
	tracen, err := traceCount(req)
	if err == nil && r.rateLimited(tracen) {
		// this payload can not be accepted
		io.Copy(io.Discard, req.Body) //nolint:errcheck
		w.WriteHeader(r.rateLimiterResponse)
		r.replyOK(req, v, w)
		ts.PayloadRefused.Inc()
		return
	}
	if err == errInvalidHeaderTraceCountValue {
		log.Errorf("Failed to count traces: %s", err)
	}

	start := time.Now()
	tp, ranHook, err := decodeTracerPayload(v, req, ts, r.containerIDProvider)
	defer func(err error) {
		tags := append(ts.AsTags(), fmt.Sprintf("success:%v", err == nil))
		metrics.Histogram("datadog.trace_agent.receiver.serve_traces_ms", float64(time.Since(start))/float64(time.Millisecond), tags, 1)
	}(err)
	if err != nil {
		httpDecodingError(err, []string{"handler:traces", fmt.Sprintf("v:%s", v)}, w)
		switch err {
		case apiutil.ErrLimitedReaderLimitReached:
			ts.TracesDropped.PayloadTooLarge.Add(tracen)
		case io.EOF, io.ErrUnexpectedEOF, msgp.ErrShortBytes:
			ts.TracesDropped.EOF.Add(tracen)
		default:
			if err, ok := err.(net.Error); ok && err.Timeout() {
				ts.TracesDropped.Timeout.Add(tracen)
			} else {
				ts.TracesDropped.DecodingError.Add(tracen)
			}
		}
		log.Errorf("Cannot decode %s traces payload: %v", v, err)
		return
	}
	if !ranHook {
		// The decoder of this request did not run the pb.MetaHook. The user is either using
		// a deprecated endpoint or Content-Type, or, a new decoder was implemented and the
		// the hook was not added.
		log.Debug("Decoded the request without running pb.MetaHook. If this is a newly implemented endpoint, please make sure to run it!")
		if _, ok := pb.MetaHook(); ok {
			log.Warn("Received request on deprecated API endpoint or Content-Type. Performance is degraded. If you think this is an error, please contact support with this message.")
			runMetaHook(tp.Chunks)
		}
	}
	if n, ok := r.replyOK(req, v, w); ok {
		tags := append(ts.AsTags(), "endpoint:traces_"+string(v))
		metrics.Histogram("datadog.trace_agent.receiver.rate_response_bytes", float64(n), tags, 1)
	}

	ts.TracesReceived.Add(int64(len(tp.Chunks)))
	ts.TracesBytes.Add(req.Body.(*apiutil.LimitedReader).Count)
	ts.PayloadAccepted.Inc()

	if ctags := getContainerTags(r.conf.ContainerTags, tp.ContainerID); ctags != "" {
		if tp.Tags == nil {
			tp.Tags = make(map[string]string)
		}
		tp.Tags[tagContainersTags] = ctags
	}

	payload := &Payload{
		Source:                 ts,
		TracerPayload:          tp,
		ClientComputedTopLevel: req.Header.Get(header.ComputedTopLevel) != "",
		ClientComputedStats:    req.Header.Get(header.ComputedStats) != "",
		ClientDroppedP0s:       droppedTracesFromHeader(req.Header, ts),
	}

	select {
	case r.out <- payload:
		// ok
	default:
		// channel blocked, add a goroutine to ensure we never drop
		r.wg.Add(1)
		count := r.outOfCPUCounter.Inc()
		if (count-1)%outOfCPULogThreshold == 0 {
			// Log a warning on the first occurrence and every n+outOfCPULogThreshold occurrences.
			log.Warnf("The Agent is falling behind on processing traces, %d extra threads have been created since the Agent started. See https://docs.datadoghq.com/tracing/troubleshooting/agent_apm_resource_usage", count)
		}
		go func() {
			metrics.Count("datadog.trace_agent.receiver.queued_send", 1, nil, 1)
			defer func() {
				r.wg.Done()
				watchdog.LogOnPanic()
			}()
			r.out <- payload
		}()
	}
}

// runMetaHook runs the pb.MetaHook on all spans from traces.
func runMetaHook(chunks []*pb.TraceChunk) {
	hook, ok := pb.MetaHook()
	if !ok {
		return
	}
	for _, chunk := range chunks {
		for _, span := range chunk.Spans {
			for k, v := range span.Meta {
				if newv := hook(k, v); newv != v {
					span.Meta[k] = newv
				}
			}
		}
	}
}

func droppedTracesFromHeader(h http.Header, ts *info.TagStats) int64 {
	var dropped int64
	if v := h.Get(header.DroppedP0Traces); v != "" {
		count, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			dropped = count
			ts.ClientDroppedP0Traces.Add(count)
		}
	}
	if v := h.Get(header.DroppedP0Spans); v != "" {
		count, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			ts.ClientDroppedP0Spans.Add(count)
		}
	}
	return dropped
}

// handleServices handle a request with a list of several services
func (r *HTTPReceiver) handleServices(v Version, w http.ResponseWriter, req *http.Request) {
	httpOK(w)

	// Do nothing, services are no longer being sent to Datadog as of July 2019
	// and are now automatically extracted from traces.
}

// loop periodically submits stats about the receiver to statsd
func (r *HTTPReceiver) loop() {
	defer close(r.exit)

	var lastLog time.Time
	accStats := info.NewReceiverStats()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	tw := time.NewTicker(r.conf.WatchdogInterval)
	defer tw.Stop()

	for {
		select {
		case <-r.exit:
			return
		case now := <-tw.C:
			r.watchdog(now)
		case now := <-t.C:
			metrics.Gauge("datadog.trace_agent.heartbeat", 1, nil, 1)
			metrics.Gauge("datadog.trace_agent.receiver.out_chan_fill", float64(len(r.out))/float64(cap(r.out)), nil, 1)

			// We update accStats with the new stats we collected
			accStats.Acc(r.Stats)

			// Publish and reset the stats accumulated during the last flush
			r.Stats.PublishAndReset()

			if now.Sub(lastLog) >= time.Minute {
				// We expose the stats accumulated to expvar
				info.UpdateReceiverStats(accStats)

				// We reset the stats accumulated during the last minute
				accStats.LogAndResetStats()
				lastLog = now

				// Also publish rates by service (they are updated by receiver)
				rates := r.dynConf.RateByService.GetNewState("").Rates
				info.UpdateRateByService(rates)
			}
		}
	}
}

// killProcess exits the process with the given msg; replaced in tests.
var killProcess = func(format string, a ...interface{}) {
	log.Criticalf(format, a...)
	os.Exit(1)
}

// watchdog checks the trace-agent's heap and CPU usage and updates the rate limiter using a correct
// sampling rate to maintain resource usage within set thresholds. These thresholds are defined by
// the configuration MaxMemory and MaxCPU. If these values are 0, all limits are disabled and the rate
// limiter will accept everything.
func (r *HTTPReceiver) watchdog(now time.Time) {
	cpu, cpuErr := watchdog.CPU(now)
	wi := watchdog.Info{
		Mem: watchdog.Mem(),
		CPU: cpu,
	}
	rateMem := 1.0
	if r.conf.MaxMemory > 0 {
		if current, allowed := float64(wi.Mem.Alloc), r.conf.MaxMemory*1.5; current > allowed {
			// This is a safety mechanism: if the agent is using more than 1.5x max. memory, there
			// is likely a leak somewhere; we'll kill the process to avoid polluting host memory.
			metrics.Count("datadog.trace_agent.receiver.oom_kill", 1, nil, 1)
			metrics.Flush()
			log.Criticalf("Killing process. Memory threshold exceeded: %.2fM / %.2fM", current/1024/1024, allowed/1024/1024)
			killProcess("OOM")
		}
		rateMem = computeRateLimitingRate(r.conf.MaxMemory, float64(wi.Mem.Alloc), r.RateLimiter.RealRate())
		if rateMem < 1 {
			log.Warnf("Memory threshold exceeded (apm_config.max_memory: %.0f bytes): %d", r.conf.MaxMemory, wi.Mem.Alloc)
		}
	}
	rateCPU := 1.0
	if r.conf.MaxCPU > 0 {
		if cpuErr != nil {
			log.Errorf("Error retrieving current CPU usage: %v. Reusing previous value", cpuErr)
		}
		rateCPU = computeRateLimitingRate(r.conf.MaxCPU, wi.CPU.UserAvg, r.RateLimiter.RealRate())
		if rateCPU < 1 {
			log.Warnf("CPU threshold exceeded (apm_config.max_cpu_percent: %.0f): %.0f", r.conf.MaxCPU*100, wi.CPU.UserAvg*100)
		}
	}

	r.RateLimiter.SetTargetRate(math.Min(rateCPU, rateMem))

	stats := r.RateLimiter.Stats()

	info.UpdateRateLimiter(*stats)
	info.UpdateWatchdogInfo(wi)

	metrics.Gauge("datadog.trace_agent.heap_alloc", float64(wi.Mem.Alloc), nil, 1)
	metrics.Gauge("datadog.trace_agent.cpu_percent", wi.CPU.UserAvg*100, nil, 1)
	metrics.Gauge("datadog.trace_agent.receiver.ratelimit", stats.TargetRate, nil, 1)
}

// Languages returns the list of the languages used in the traces the agent receives.
func (r *HTTPReceiver) Languages() string {
	// We need to use this map because we can have several tags for a same language.
	langs := make(map[string]bool)
	str := []string{}

	r.Stats.RLock()
	for tags := range r.Stats.Stats {
		if _, ok := langs[tags.Lang]; !ok {
			str = append(str, tags.Lang)
			langs[tags.Lang] = true
		}
	}
	r.Stats.RUnlock()

	sort.Strings(str)
	return strings.Join(str, "|")
}

// decodeRequest decodes the payload in http request `req` into `dest`.
// It handles only v02, v03, v04 requests.
// - ranHook reports whether the decoder was able to run the pb.MetaHook
// - err is the first error encountered
func decodeRequest(req *http.Request, dest *pb.Traces) (ranHook bool, err error) {
	switch mediaType := getMediaType(req); mediaType {
	case "application/msgpack":
		buf := getBuffer()
		defer putBuffer(buf)
		_, err = io.Copy(buf, req.Body)
		if err != nil {
			return false, err
		}
		_, err = dest.UnmarshalMsg(buf.Bytes())
		return true, err
	case "application/json":
		fallthrough
	case "text/json":
		fallthrough
	case "":
		err = json.NewDecoder(req.Body).Decode(&dest)
		return false, err
	default:
		// do our best
		if err1 := json.NewDecoder(req.Body).Decode(&dest); err1 != nil {
			buf := getBuffer()
			defer putBuffer(buf)
			_, err2 := io.Copy(buf, req.Body)
			if err2 != nil {
				return false, err2
			}
			_, err2 = dest.UnmarshalMsg(buf.Bytes())
			return true, err2
		}
		return false, nil
	}
}

func traceChunksFromSpans(spans []pb.Span) []*pb.TraceChunk {
	traceChunks := []*pb.TraceChunk{}
	byID := make(map[uint64][]*pb.Span)
	for _, s := range spans {
		byID[s.TraceID] = append(byID[s.TraceID], &s)
	}
	for _, t := range byID {
		traceChunks = append(traceChunks, &pb.TraceChunk{
			Priority: int32(sampler.PriorityNone),
			Spans:    t,
		})
	}
	return traceChunks
}

func traceChunksFromTraces(traces pb.Traces) []*pb.TraceChunk {
	traceChunks := make([]*pb.TraceChunk, 0, len(traces))
	for _, trace := range traces {
		traceChunks = append(traceChunks, &pb.TraceChunk{
			Priority: int32(sampler.PriorityNone),
			Spans:    trace,
		})
	}

	return traceChunks
}

// getContainerTag returns container and orchestrator tags belonging to containerID. If containerID
// is empty or no tags are found, an empty string is returned.
func getContainerTags(fn func(string) ([]string, error), containerID string) string {
	if containerID == "" {
		return ""
	}
	if fn == nil {
		log.Warn("ContainerTags not configured")
		return ""
	}
	list, err := fn(containerID)
	if err != nil {
		log.Tracef("Getting container tags for ID %q: %v", containerID, err)
		return ""
	}
	log.Tracef("Getting container tags for ID %q: %v", containerID, list)
	return strings.Join(list, ",")
}

// getMediaType attempts to return the media type from the Content-Type MIME header. If it fails
// it returns the default media type "application/json".
func getMediaType(req *http.Request) string {
	mt, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		log.Debugf(`Error parsing media type: %v, assuming "application/json"`, err)
		return "application/json"
	}
	return mt
}
