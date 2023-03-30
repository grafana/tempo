// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package writer

import (
	"compress/gzip"
	"errors"
	"io"
	"math"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics/timing"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/telemetry"

	"github.com/tinylib/msgp/msgp"
)

// pathStats is the target host API path for delivering stats.
const pathStats = "/api/v0.2/stats"

const (
	// bytesPerEntry specifies the approximate size an entry in a stat payload occupies.
	bytesPerEntry = 375
	// maxEntriesPerPayload is the maximum number of entries in a stat payload. An
	// entry has an average size of 375 bytes in a compressed payload. The current
	// Datadog intake API limits a compressed payload to ~3MB (8,000 entries), but
	// let's have the default ensure we don't have paylods > 1.5 MB (4,000
	// entries).
	maxEntriesPerPayload = 4000
)

// StatsWriter ingests stats buckets and flushes them to the API.
type StatsWriter struct {
	in      <-chan pb.StatsPayload
	senders []*sender
	stop    chan struct{}
	stats   *info.StatsWriterInfo
	conf    *config.AgentConfig

	// syncMode reports whether the writer should flush on its own or only when FlushSync is called
	syncMode  bool
	payloads  []pb.StatsPayload // payloads buffered for sync mode
	flushChan chan chan struct{}

	easylog *log.ThrottledLogger
}

// NewStatsWriter returns a new StatsWriter. It must be started using Run.
func NewStatsWriter(cfg *config.AgentConfig, in <-chan pb.StatsPayload, telemetryCollector telemetry.TelemetryCollector) *StatsWriter {
	sw := &StatsWriter{
		in:        in,
		stats:     &info.StatsWriterInfo{},
		stop:      make(chan struct{}),
		flushChan: make(chan chan struct{}),
		syncMode:  cfg.SynchronousFlushing,
		easylog:   log.NewThrottled(5, 10*time.Second), // no more than 5 messages every 10 seconds
		conf:      cfg,
	}
	climit := cfg.StatsWriter.ConnectionLimit
	if climit == 0 {
		// Allow 1% of the connection limit to outgoing sends. The original
		// connection limit was removed and used to be 2000 (1% = 20)
		climit = 20
	}
	qsize := cfg.StatsWriter.QueueSize
	if qsize == 0 {
		payloadSize := float64(maxEntriesPerPayload * bytesPerEntry)
		// default to 25% of maximum memory.
		maxmem := cfg.MaxMemory / 4
		if maxmem == 0 {
			// or 250MB if unbound
			maxmem = 250 * 1024 * 1024
		}
		qsize = int(math.Max(1, maxmem/payloadSize))
	}
	log.Debugf("Stats writer initialized (climit=%d qsize=%d)", climit, qsize)
	sw.senders = newSenders(cfg, sw, pathStats, climit, qsize, telemetryCollector)
	return sw
}

// Run starts the StatsWriter, making it ready to receive stats and report metrics.
func (w *StatsWriter) Run() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	defer close(w.stop)
	for {
		select {
		case stats := <-w.in:
			w.addStats(stats)
			if !w.syncMode {
				w.sendPayloads()
			}
		case notify := <-w.flushChan:
			w.sendPayloads()
			notify <- struct{}{}
		case <-t.C:
			w.report()
		case <-w.stop:
			return
		}
	}
}

// FlushSync blocks and sends pending payloads when syncMode is true
func (w *StatsWriter) FlushSync() error {
	if !w.syncMode {
		return errors.New("not flushing; sync mode not enabled")
	}

	defer w.report()
	notify := make(chan struct{}, 1)
	w.flushChan <- notify
	<-notify
	return nil
}

// Stop stops a running StatsWriter.
func (w *StatsWriter) Stop() {
	w.stop <- struct{}{}
	<-w.stop
	stopSenders(w.senders)
}

func (w *StatsWriter) addStats(sp pb.StatsPayload) {
	defer timing.Since("datadog.trace_agent.stats_writer.encode_ms", time.Now())
	payloads := w.buildPayloads(sp, maxEntriesPerPayload)
	w.payloads = append(w.payloads, payloads...)
}

// SendPayload sends a stats payload to the Datadog backend.
func (w *StatsWriter) SendPayload(p pb.StatsPayload) {
	req := newPayload(map[string]string{
		headerLanguages:    strings.Join(info.Languages(), "|"),
		"Content-Type":     "application/msgpack",
		"Content-Encoding": "gzip",
	})
	if err := encodePayload(req.body, p); err != nil {
		log.Errorf("Stats encoding error: %v", err)
		return
	}
	sendPayloads(w.senders, req, w.syncMode)
}

func (w *StatsWriter) sendPayloads() {
	for _, p := range w.payloads {
		w.SendPayload(p)
	}
	w.resetBuffer()
}

func (w *StatsWriter) resetBuffer() {
	w.payloads = make([]pb.StatsPayload, 0, len(w.payloads))
}

// encodePayload encodes the payload as Gzipped msgPack into w.
func encodePayload(w io.Writer, payload pb.StatsPayload) error {
	gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		return err
	}
	defer func() {
		if err := gz.Close(); err != nil {
			log.Errorf("Error closing gzip stream when writing stats payload: %v", err)
		}
	}()
	return msgp.Encode(gz, &payload)
}

// buildPayloads splits pb.ClientStatsPayload that have more than maxEntriesPerPayload
// and then groups them into pb.StatsPayload with less than maxEntriesPerPayload
func (w *StatsWriter) buildPayloads(sp pb.StatsPayload, maxEntriesPerPayload int) []pb.StatsPayload {
	split := splitPayloads(sp.Stats, maxEntriesPerPayload)
	grouped := make([]pb.StatsPayload, 0, len(sp.Stats))
	current := pb.StatsPayload{
		AgentHostname:  sp.AgentHostname,
		AgentEnv:       sp.AgentEnv,
		AgentVersion:   sp.AgentVersion,
		ClientComputed: sp.ClientComputed,
	}
	var nbEntries, nbBuckets int
	addPayload := func() {
		log.Debugf("Flushing %d entries (buckets=%d client_payloads=%d)", nbEntries, nbBuckets, len(current.Stats))
		w.stats.StatsBuckets.Add(int64(nbBuckets))
		w.stats.ClientPayloads.Add(int64(len(current.Stats)))
		w.stats.StatsEntries.Add(int64(nbEntries))
		grouped = append(grouped, current)
		current.Stats = nil
		nbEntries = 0
		nbBuckets = 0
	}
	for _, p := range split {
		if nbEntries+p.nbEntries > maxEntriesPerPayload {
			addPayload()
		}
		nbEntries += p.nbEntries
		nbBuckets += len(p.Stats)
		w.resolveContainerTags(&p.ClientStatsPayload)
		current.Stats = append(current.Stats, p.ClientStatsPayload)
	}
	if nbEntries > 0 {
		addPayload()
	}
	if len(grouped) > 1 {
		w.stats.Splits.Inc()
	}
	return grouped
}

// resolveContainerTags takes any ContainerID found in p to fill in the appropriate tags.
func (w *StatsWriter) resolveContainerTags(p *pb.ClientStatsPayload) {
	if p.ContainerID == "" {
		p.Tags = nil
		return
	}
	ctags, err := w.conf.ContainerTags(p.ContainerID)
	switch {
	case err != nil:
		log.Tracef("Error resolving container tags for %q: %v", p.ContainerID, err)
		p.ContainerID = ""
		p.Tags = nil
	case len(ctags) == 0:
		p.Tags = nil
	default:
		p.Tags = ctags
	}
}

func splitPayloads(payloads []pb.ClientStatsPayload, maxEntriesPerPayload int) []clientStatsPayload {
	split := make([]clientStatsPayload, 0, len(payloads))
	for _, p := range payloads {
		split = append(split, splitPayload(p, maxEntriesPerPayload)...)
	}
	return split
}

type timeWindow struct{ start, duration uint64 }

type clientStatsPayload struct {
	pb.ClientStatsPayload
	nbEntries int
	// bucketIndexes maps from a timeWindow to a bucket in the ClientStatsPayload.
	// it allows quick checking of what bucket to add a payload to.
	bucketIndexes map[timeWindow]int
}

// splitPayload splits a stats payload to ensure that each stats payload has less than maxEntriesPerPayload entries.
func splitPayload(p pb.ClientStatsPayload, maxEntriesPerPayload int) []clientStatsPayload {
	if len(p.Stats) == 0 {
		return nil
	}
	// 1. Get how many payloads we need, based on the total number of entries.
	nbEntries := 0
	for _, b := range p.Stats {
		nbEntries += len(b.Stats)
	}
	if maxEntriesPerPayload <= 0 || nbEntries < maxEntriesPerPayload {
		// nothing to do, break early
		return []clientStatsPayload{{ClientStatsPayload: p, nbEntries: nbEntries}}
	}
	nbPayloads := nbEntries / maxEntriesPerPayload
	if nbEntries%maxEntriesPerPayload != 0 {
		nbPayloads++
	}

	// 2. Initialize a slice of nbPayloads indexes maps, mapping a time window (stat +
	//    duration) to a stats payload.
	payloads := make([]clientStatsPayload, nbPayloads)
	for i := 0; i < nbPayloads; i++ {
		payloads[i] = clientStatsPayload{
			bucketIndexes: make(map[timeWindow]int, 1),
			ClientStatsPayload: pb.ClientStatsPayload{
				Hostname:         p.Hostname,
				Env:              p.Env,
				Version:          p.Version,
				Service:          p.Service,
				Lang:             p.Lang,
				TracerVersion:    p.TracerVersion,
				RuntimeID:        p.RuntimeID,
				Sequence:         p.Sequence,
				AgentAggregation: p.AgentAggregation,
				ContainerID:      p.ContainerID,
				Stats:            make([]pb.ClientStatsBucket, 0, maxEntriesPerPayload),
			},
		}
	}
	// 3. Iterate over all entries of each stats. Add the entry to one of
	//    the payloads, in a round robin fashion. Use the bucketIndexes map to
	//    ensure that we have one ClientStatsBucket per timeWindow for each ClientStatsPayload.
	i := 0
	for _, b := range p.Stats {
		tw := timeWindow{b.Start, b.Duration}
		for _, g := range b.Stats {
			j := i % nbPayloads
			bi, ok := payloads[j].bucketIndexes[tw]
			if !ok {
				bi = len(payloads[j].Stats)
				payloads[j].bucketIndexes[tw] = bi
				payloads[j].Stats = append(payloads[j].Stats, pb.ClientStatsBucket{Start: tw.start, Duration: tw.duration})
			}
			// here, we can just append the group, because there are no duplicate groups in the original stats payloads sent to the writer.
			payloads[j].Stats[bi].Stats = append(payloads[j].Stats[bi].Stats, g)
			payloads[j].nbEntries++
			i++
		}
	}
	return payloads
}

var _ eventRecorder = (*StatsWriter)(nil)

func (w *StatsWriter) report() {
	metrics.Count("datadog.trace_agent.stats_writer.client_payloads", w.stats.ClientPayloads.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.payloads", w.stats.Payloads.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.stats_buckets", w.stats.StatsBuckets.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.stats_entries", w.stats.StatsEntries.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.bytes", w.stats.Bytes.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.retries", w.stats.Retries.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.splits", w.stats.Splits.Swap(0), nil, 1)
	metrics.Count("datadog.trace_agent.stats_writer.errors", w.stats.Errors.Swap(0), nil, 1)
}

// recordEvent implements eventRecorder.
func (w *StatsWriter) recordEvent(t eventType, data *eventData) {
	if data != nil {
		metrics.Histogram("datadog.trace_agent.stats_writer.connection_fill", data.connectionFill, nil, 1)
		metrics.Histogram("datadog.trace_agent.stats_writer.queue_fill", data.queueFill, nil, 1)
	}
	switch t {
	case eventTypeRetry:
		log.Debugf("Retrying to flush stats payload (error: %q)", data.err)
		w.stats.Retries.Inc()

	case eventTypeSent:
		log.Debugf("Flushed stats to the API; time: %s, bytes: %d", data.duration, data.bytes)
		timing.Since("datadog.trace_agent.stats_writer.flush_duration", time.Now().Add(-data.duration))
		w.stats.Bytes.Add(int64(data.bytes))
		w.stats.Payloads.Inc()

	case eventTypeRejected:
		log.Warnf("Stats writer payload rejected by edge: %v", data.err)
		w.stats.Errors.Inc()

	case eventTypeDropped:
		w.easylog.Warn("Stats writer queue full. Payload dropped (%.2fKB).", float64(data.bytes)/1024)
		metrics.Count("datadog.trace_agent.stats_writer.dropped", 1, nil, 1)
		metrics.Count("datadog.trace_agent.stats_writer.dropped_bytes", int64(data.bytes), nil, 1)
	}
}
