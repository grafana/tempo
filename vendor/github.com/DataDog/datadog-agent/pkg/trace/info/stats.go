// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package info

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/atomic"

	"github.com/DataDog/datadog-agent/pkg/trace/sampler"

	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
)

// ReceiverStats is used to store all the stats per tags.
type ReceiverStats struct {
	sync.RWMutex
	Stats map[Tags]*TagStats
}

// NewReceiverStats returns a new ReceiverStats
func NewReceiverStats() *ReceiverStats {
	return &ReceiverStats{sync.RWMutex{}, map[Tags]*TagStats{}}
}

// GetTagStats returns the struct in which the stats will be stored depending of their tags.
func (rs *ReceiverStats) GetTagStats(tags Tags) *TagStats {
	rs.Lock()
	tagStats, ok := rs.Stats[tags]
	if !ok {
		tagStats = newTagStats(tags)
		rs.Stats[tags] = tagStats
	}
	rs.Unlock()

	return tagStats
}

// Acc accumulates the stats from another ReceiverStats struct.
func (rs *ReceiverStats) Acc(recent *ReceiverStats) {
	recent.Lock()
	for _, tagStats := range recent.Stats {
		ts := rs.GetTagStats(tagStats.Tags)
		ts.update(&tagStats.Stats)
	}
	recent.Unlock()
}

// PublishAndReset updates stats about per-tag stats
func (rs *ReceiverStats) PublishAndReset() {
	rs.RLock()
	for _, tagStats := range rs.Stats {
		tagStats.publishAndReset()
	}
	rs.RUnlock()
}

// Languages returns the set of languages reporting traces to the Agent.
func (rs *ReceiverStats) Languages() []string {
	langSet := make(map[string]bool)
	langs := []string{}

	rs.RLock()
	for tags := range rs.Stats {
		if _, ok := langSet[tags.Lang]; !ok {
			langs = append(langs, tags.Lang)
			langSet[tags.Lang] = true
		}
	}
	rs.RUnlock()

	sort.Strings(langs)

	return langs
}

// LogAndResetStats logs one-line summaries of ReceiverStats and resets internal data. Problematic stats are logged as warnings.
func (rs *ReceiverStats) LogAndResetStats() {
	rs.Lock()
	defer rs.Unlock()

	if len(rs.Stats) == 0 {
		log.Info("No data received")
		return
	}

	for k, ts := range rs.Stats {
		if !ts.isEmpty() {
			tags := ts.Tags.toArray()
			log.Infof("%v -> %s\n", tags, ts.infoString())
			warnString := ts.WarnString()
			if len(warnString) > 0 {
				log.Warnf("%v -> %s. Enable debug logging for more details.\n", tags, warnString)
			}
		}
		delete(rs.Stats, k)
	}
}

// TagStats is the struct used to associate the stats with their set of tags.
type TagStats struct {
	Tags
	Stats
}

func newTagStats(tags Tags) *TagStats {
	return &TagStats{Tags: tags, Stats: NewStats()}
}

// AsTags returns all the tags contained in the TagStats.
func (ts *TagStats) AsTags() []string {
	return (&ts.Tags).toArray()
}

func (ts *TagStats) publishAndReset() {
	// Atomically load and reset any metrics used multiple times from ts
	tracesReceived := ts.TracesReceived.Swap(0)

	// Publish the stats
	tags := ts.Tags.toArray()

	metrics.Count("datadog.trace_agent.receiver.trace", tracesReceived, tags, 1)
	metrics.Count("datadog.trace_agent.receiver.traces_received", tracesReceived, tags, 1)
	metrics.Count("datadog.trace_agent.receiver.traces_filtered",
		ts.TracesFiltered.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.traces_priority",
		ts.TracesPriorityNone.Swap(0), append(tags, "priority:none"), 1)
	metrics.Count("datadog.trace_agent.receiver.traces_bytes",
		ts.TracesBytes.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.spans_received",
		ts.SpansReceived.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.spans_dropped",
		ts.SpansDropped.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.spans_filtered",
		ts.SpansFiltered.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.events_extracted",
		ts.EventsExtracted.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.events_sampled",
		ts.EventsSampled.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.payload_accepted",
		ts.PayloadAccepted.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.payload_refused",
		ts.PayloadRefused.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.client_dropped_p0_spans",
		ts.ClientDroppedP0Spans.Swap(0), tags, 1)
	metrics.Count("datadog.trace_agent.receiver.client_dropped_p0_traces",
		ts.ClientDroppedP0Traces.Swap(0), tags, 1)

	for reason, counter := range ts.TracesDropped.tagCounters() {
		metrics.Count("datadog.trace_agent.normalizer.traces_dropped",
			counter.Swap(0), append(tags, "reason:"+reason), 1)
	}

	for reason, counter := range ts.SpansMalformed.tagCounters() {
		metrics.Count("datadog.trace_agent.normalizer.spans_malformed",
			counter.Swap(0), append(tags, "reason:"+reason), 1)
	}

	for priority, counter := range ts.TracesPerSamplingPriority.tagCounters() {
		count := counter.Swap(0)
		if count > 0 {
			metrics.Count("datadog.trace_agent.receiver.traces_priority",
				count, append(tags, "priority:"+priority), 1)
		}
	}
}

// mapToString serializes the entries in this map into format "key1: value1, key2: value2, ...", sorted by
// key to ensure consistent output order. Only non-zero values are included.
func mapToString(m map[string]int64) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var results []string
	for _, key := range keys {
		value := m[key]
		if value > 0 {
			results = append(results, fmt.Sprintf("%s:%d", key, value))
		}
	}
	return strings.Join(results, ", ")
}

// TracesDropped contains counts for reasons traces have been dropped
type TracesDropped struct {
	// all atomic values are included as values in this struct, to simplify umarshaling and
	// initialization of the type.  The atomic values _must_ occur first in the struct.

	// DecodingError is when the agent fails to decode a trace payload
	DecodingError atomic.Int64
	// PayloadTooLarge specifies the number of traces dropped due to the payload
	// being too large to be accepted.
	PayloadTooLarge atomic.Int64
	// EmptyTrace is when the trace contains no spans
	EmptyTrace atomic.Int64
	// TraceIDZero is when any spans in a trace have TraceId=0
	TraceIDZero atomic.Int64
	// SpanIDZero is when any span has SpanId=0
	SpanIDZero atomic.Int64
	// ForeignSpan is when a span in a trace has a TraceId that is different than the first span in the trace
	ForeignSpan atomic.Int64
	// Timeout is when a request times out.
	Timeout atomic.Int64
	// EOF is when an unexpected EOF is encountered, this can happen because the client has aborted
	// or because a bad payload (i.e. shorter than claimed in Content-Length) was sent.
	EOF atomic.Int64
}

func (s *TracesDropped) tagCounters() map[string]*atomic.Int64 {
	return map[string]*atomic.Int64{
		"payload_too_large": &s.PayloadTooLarge,
		"decoding_error":    &s.DecodingError,
		"empty_trace":       &s.EmptyTrace,
		"trace_id_zero":     &s.TraceIDZero,
		"span_id_zero":      &s.SpanIDZero,
		"foreign_span":      &s.ForeignSpan,
		"timeout":           &s.Timeout,
		"unexpected_eof":    &s.EOF,
	}
}

// tagValues converts TracesDropped into a map representation with keys matching standardized names for all reasons
func (s *TracesDropped) tagValues() map[string]int64 {
	v := make(map[string]int64)
	for tag, counter := range s.tagCounters() {
		v[tag] = counter.Load()
	}
	return v
}

func (s *TracesDropped) String() string {
	return mapToString(s.tagValues())
}

// SpansMalformed contains counts for reasons malformed spans have been accepted after applying automatic fixes
type SpansMalformed struct {
	// all atomic values are included as values in this struct, to simplify umarshaling and
	// initialization of the type.  The atomic values _must_ occur first in the struct.

	// DuplicateSpanID is when one or more spans in a trace have the same SpanId
	DuplicateSpanID atomic.Int64
	// ServiceEmpty is when a span has an empty Service field
	ServiceEmpty atomic.Int64
	// ServiceTruncate is when a span's Service is truncated for exceeding the max length
	ServiceTruncate atomic.Int64
	// ServiceInvalid is when a span's Service doesn't conform to Datadog tag naming standards
	ServiceInvalid atomic.Int64
	// SpanNameEmpty is when a span's Name is empty
	SpanNameEmpty atomic.Int64
	// SpanNameTruncate is when a span's Name is truncated for exceeding the max length
	SpanNameTruncate atomic.Int64
	// SpanNameInvalid is when a span's Name doesn't conform to Datadog tag naming standards
	SpanNameInvalid atomic.Int64
	// ResourceEmpty is when a span's Resource is empty
	ResourceEmpty atomic.Int64
	// TypeTruncate is when a span's Type is truncated for exceeding the max length
	TypeTruncate atomic.Int64
	// InvalidStartDate is when a span's Start date is invalid
	InvalidStartDate atomic.Int64
	// InvalidDuration is when a span's Duration is invalid
	InvalidDuration atomic.Int64
	// InvalidHTTPStatusCode is when a span's metadata contains an invalid http status code
	InvalidHTTPStatusCode atomic.Int64
}

func (s *SpansMalformed) tagCounters() map[string]*atomic.Int64 {
	return map[string]*atomic.Int64{
		"duplicate_span_id":        &s.DuplicateSpanID,
		"service_empty":            &s.ServiceEmpty,
		"service_truncate":         &s.ServiceTruncate,
		"service_invalid":          &s.ServiceInvalid,
		"span_name_empty":          &s.SpanNameEmpty,
		"span_name_truncate":       &s.SpanNameTruncate,
		"span_name_invalid":        &s.SpanNameInvalid,
		"resource_empty":           &s.ResourceEmpty,
		"type_truncate":            &s.TypeTruncate,
		"invalid_start_date":       &s.InvalidStartDate,
		"invalid_duration":         &s.InvalidDuration,
		"invalid_http_status_code": &s.InvalidHTTPStatusCode,
	}
}

// tagValues converts SpansMalformed into a map representation with keys matching standardized names for all reasons
func (s *SpansMalformed) tagValues() map[string]int64 {
	v := make(map[string]int64)
	for reason, counter := range s.tagCounters() {
		v[reason] = counter.Load()
	}
	return v
}

func (s *SpansMalformed) String() string {
	return mapToString(s.tagValues())
}

// maxAbsPriority specifies the absolute maximum priority for stats purposes. For example, with a value
// of 10, the range of priorities reported will be [-10, 10].
const maxAbsPriority = 10

// samplingPriorityStats holds the sampling priority metrics that will be reported every 10s by the agent.
type samplingPriorityStats struct {
	// counts holds counters for each priority in position maxAbsPriorityValue + priority.
	// Priority values are expected to be in the range [-10, 10].
	counts [maxAbsPriority*2 + 1]atomic.Int64
}

// CountSamplingPriority increments the counter of observed traces with the given sampling priority by 1
func (s *samplingPriorityStats) CountSamplingPriority(p sampler.SamplingPriority) {
	if p >= (-1*maxAbsPriority) && p <= maxAbsPriority {
		s.counts[maxAbsPriority+p].Inc()
	}
}

// reset sets stats to 0
func (s *samplingPriorityStats) reset() {
	for i := range s.counts {
		s.counts[i].Store(0)
	}
}

// update absorbs recent stats on top of existing ones.
func (s *samplingPriorityStats) update(recent *samplingPriorityStats) {
	for i := range s.counts {
		s.counts[i].Add(recent.counts[i].Load())
	}
}

func (s *samplingPriorityStats) tagCounters() map[string]*atomic.Int64 {
	stats := make(map[string]*atomic.Int64)
	for i := range s.counts {
		stats[strconv.Itoa(i-maxAbsPriority)] = &s.counts[i]
	}
	return stats
}

// TagValues returns a map with the number of traces that have been observed for each priority tag
func (s *samplingPriorityStats) TagValues() map[string]int64 {
	stats := make(map[string]int64)
	for i := range s.counts {
		count := s.counts[i].Load()
		if count > 0 {
			stats[strconv.Itoa(i-maxAbsPriority)] = count
		}
	}
	return stats
}

// Stats holds the metrics that will be reported every 10s by the agent.
// Its fields require to be accessed in an atomic way.
//
// Use NewStats to initialise.
type Stats struct {
	// all atomic values are included as values in this struct, to simplify umarshaling and
	// initialization of the type.  The atomic values _must_ occur first in the struct.

	// TracesReceived is the total number of traces received, including the dropped ones.
	TracesReceived atomic.Int64
	// TracesFiltered is the number of traces filtered.
	TracesFiltered atomic.Int64
	// TracesPriorityNone is the number of traces with no sampling priority.
	TracesPriorityNone atomic.Int64
	// TracesPerPriority holds counters for each priority in position MaxAbsPriorityValue + priority.
	TracesPerSamplingPriority samplingPriorityStats
	// ClientDroppedP0Traces number of P0 traces dropped by client.
	ClientDroppedP0Traces atomic.Int64
	// ClientDroppedP0Spans number of P0 spans dropped by client.
	ClientDroppedP0Spans atomic.Int64
	// TracesBytes is the amount of data received on the traces endpoint (raw data, encoded, compressed).
	TracesBytes atomic.Int64
	// SpansReceived is the total number of spans received, including the dropped ones.
	SpansReceived atomic.Int64
	// SpansDropped is the number of spans dropped.
	SpansDropped atomic.Int64
	// SpansFiltered is the number of spans filtered.
	SpansFiltered atomic.Int64
	// EventsExtracted is the total number of APM events extracted from traces.
	EventsExtracted atomic.Int64
	// EventsSampled is the total number of APM events sampled.
	EventsSampled atomic.Int64
	// PayloadAccepted counts the number of payloads that have been accepted by the HTTP handler.
	PayloadAccepted atomic.Int64
	// PayloadRefused counts the number of payloads that have been rejected by the rate limiter.
	PayloadRefused atomic.Int64
	// TracesDropped contains stats about the count of dropped traces by reason
	TracesDropped *TracesDropped
	// SpansMalformed contains stats about the count of malformed traces by reason
	SpansMalformed *SpansMalformed
}

// NewStats returns new, ready to use stats.
func NewStats() Stats {
	return Stats{
		TracesDropped:  new(TracesDropped),
		SpansMalformed: new(SpansMalformed),
	}
}

func (s *Stats) update(recent *Stats) {
	s.TracesReceived.Add(recent.TracesReceived.Load())
	s.TracesDropped.DecodingError.Add(recent.TracesDropped.DecodingError.Load())
	s.TracesDropped.EmptyTrace.Add(recent.TracesDropped.EmptyTrace.Load())
	s.TracesDropped.TraceIDZero.Add(recent.TracesDropped.TraceIDZero.Load())
	s.TracesDropped.SpanIDZero.Add(recent.TracesDropped.SpanIDZero.Load())
	s.TracesDropped.ForeignSpan.Add(recent.TracesDropped.ForeignSpan.Load())
	s.TracesDropped.PayloadTooLarge.Add(recent.TracesDropped.PayloadTooLarge.Load())
	s.TracesDropped.Timeout.Add(recent.TracesDropped.Timeout.Load())
	s.TracesDropped.EOF.Add(recent.TracesDropped.EOF.Load())
	s.SpansMalformed.DuplicateSpanID.Add(recent.SpansMalformed.DuplicateSpanID.Load())
	s.SpansMalformed.ServiceEmpty.Add(recent.SpansMalformed.ServiceEmpty.Load())
	s.SpansMalformed.ServiceTruncate.Add(recent.SpansMalformed.ServiceTruncate.Load())
	s.SpansMalformed.ServiceInvalid.Add(recent.SpansMalformed.ServiceInvalid.Load())
	s.SpansMalformed.SpanNameEmpty.Add(recent.SpansMalformed.SpanNameEmpty.Load())
	s.SpansMalformed.SpanNameTruncate.Add(recent.SpansMalformed.SpanNameTruncate.Load())
	s.SpansMalformed.SpanNameInvalid.Add(recent.SpansMalformed.SpanNameInvalid.Load())
	s.SpansMalformed.ResourceEmpty.Add(recent.SpansMalformed.ResourceEmpty.Load())
	s.SpansMalformed.TypeTruncate.Add(recent.SpansMalformed.TypeTruncate.Load())
	s.SpansMalformed.InvalidStartDate.Add(recent.SpansMalformed.InvalidStartDate.Load())
	s.SpansMalformed.InvalidDuration.Add(recent.SpansMalformed.InvalidDuration.Load())
	s.SpansMalformed.InvalidHTTPStatusCode.Add(recent.SpansMalformed.InvalidHTTPStatusCode.Load())
	s.TracesFiltered.Add(recent.TracesFiltered.Load())
	s.TracesPriorityNone.Add(recent.TracesPriorityNone.Load())
	s.ClientDroppedP0Traces.Add(recent.ClientDroppedP0Traces.Load())
	s.ClientDroppedP0Spans.Add(recent.ClientDroppedP0Spans.Load())
	s.TracesBytes.Add(recent.TracesBytes.Load())
	s.SpansReceived.Add(recent.SpansReceived.Load())
	s.SpansDropped.Add(recent.SpansDropped.Load())
	s.SpansFiltered.Add(recent.SpansFiltered.Load())
	s.EventsExtracted.Add(recent.EventsExtracted.Load())
	s.EventsSampled.Add(recent.EventsSampled.Load())
	s.PayloadAccepted.Add(recent.PayloadAccepted.Load())
	s.PayloadRefused.Add(recent.PayloadRefused.Load())
	s.TracesPerSamplingPriority.update(&recent.TracesPerSamplingPriority)
}

func (s *Stats) isEmpty() bool {
	tracesBytes := s.TracesBytes.Load()

	return tracesBytes == 0
}

// infoString returns a string representation of the Stats struct containing standard operational stats (not problems)
func (s *Stats) infoString() string {
	// Atomically load the stats
	tracesReceived := s.TracesReceived.Load()
	tracesFiltered := s.TracesFiltered.Load()
	// Omitting priority information, use expvar or metrics for debugging purpose
	tracesBytes := s.TracesBytes.Load()
	eventsExtracted := s.EventsExtracted.Load()
	eventsSampled := s.EventsSampled.Load()

	return fmt.Sprintf("traces received: %d, traces filtered: %d, "+
		"traces amount: %d bytes, events extracted: %d, events sampled: %d",
		tracesReceived, tracesFiltered, tracesBytes, eventsExtracted, eventsSampled)
}

// WarnString returns a string representation of the Stats struct containing only issues which we should be warning on
// if there are no issues then an empty string is returned
func (ts *TagStats) WarnString() string {
	var (
		w []string
		d string
	)
	if ts.TracesDropped != nil {
		d = ts.TracesDropped.String()
	}
	if len(d) > 0 {
		w = append(w, fmt.Sprintf("traces_dropped(%s)", d))
	}
	var m string
	if ts.SpansMalformed != nil {
		m = ts.SpansMalformed.String()
	}
	if len(m) > 0 {
		w = append(w, fmt.Sprintf("spans_malformed(%s)", m))
	}
	return strings.Join(w, ", ")
}

// Tags holds the tags we parse when we handle the header of the payload.
type Tags struct {
	Lang, LangVersion, LangVendor, Interpreter, TracerVersion string
	EndpointVersion                                           string
}

// toArray will transform the Tags struct into a slice of string.
// We only publish the non-empty tags.
func (t *Tags) toArray() []string {
	tags := make([]string, 0, 5)

	if t.Lang != "" {
		tags = append(tags, "lang:"+t.Lang)
	}
	if t.LangVersion != "" {
		tags = append(tags, "lang_version:"+t.LangVersion)
	}
	if t.LangVendor != "" {
		tags = append(tags, "lang_vendor:"+t.LangVendor)
	}
	if t.Interpreter != "" {
		tags = append(tags, "interpreter:"+t.Interpreter)
	}
	if t.TracerVersion != "" {
		tags = append(tags, "tracer_version:"+t.TracerVersion)
	}
	if t.EndpointVersion != "" {
		tags = append(tags, "endpoint_version:"+t.EndpointVersion)
	}

	return tags
}
