package traceql

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
)

type MetadataCombiner struct {
	trs map[string]*tempopb.TraceSearchMetadata
}

func NewMetadataCombiner() *MetadataCombiner {
	return &MetadataCombiner{
		trs: make(map[string]*tempopb.TraceSearchMetadata),
	}
}

// AddMetadata adds the new metadata to the map. if it already exists
// use CombineSearchResults to combine the two
func (c *MetadataCombiner) AddMetadata(new *tempopb.TraceSearchMetadata) {
	if existing, ok := c.trs[new.TraceID]; ok {
		combineSearchResults(existing, new)
		return
	}

	c.trs[new.TraceID] = new
}

func (c *MetadataCombiner) Count() int {
	return len(c.trs)
}

func (c *MetadataCombiner) Exists(id string) bool {
	_, ok := c.trs[id]
	return ok
}

func (c *MetadataCombiner) Metadata() []*tempopb.TraceSearchMetadata {
	m := make([]*tempopb.TraceSearchMetadata, 0, len(c.trs))
	for _, tr := range c.trs {
		m = append(m, tr)
	}
	sort.Slice(m, func(i, j int) bool {
		return m[i].StartTimeUnixNano > m[j].StartTimeUnixNano
	})
	return m
}

// combineSearchResults overlays the incoming search result with the existing result. This is required
// for the following reason:  a trace may be present in multiple blocks, or in partial segments
// in live traces.  The results should reflect elements of all segments.
func combineSearchResults(existing *tempopb.TraceSearchMetadata, incoming *tempopb.TraceSearchMetadata) {
	if existing.TraceID == "" {
		existing.TraceID = incoming.TraceID
	}

	if existing.RootServiceName == "" {
		existing.RootServiceName = incoming.RootServiceName
	}

	if existing.RootTraceName == "" {
		existing.RootTraceName = incoming.RootTraceName
	}

	// Earliest start time.
	if existing.StartTimeUnixNano > incoming.StartTimeUnixNano || existing.StartTimeUnixNano == 0 {
		existing.StartTimeUnixNano = incoming.StartTimeUnixNano
	}

	// Longest duration
	if existing.DurationMs < incoming.DurationMs || existing.DurationMs == 0 {
		existing.DurationMs = incoming.DurationMs
	}

	// make a map of existing Spansets
	existingSS := make(map[string]*tempopb.SpanSet)
	for _, ss := range existing.SpanSets {
		existingSS[spansetID(ss)] = ss
	}

	// add any new spansets
	for _, ss := range incoming.SpanSets {
		id := spansetID(ss)
		// if not found just add directly
		if _, ok := existingSS[id]; !ok {
			existing.SpanSets = append(existing.SpanSets, ss)
			continue
		}

		// otherwise combine with existing
		combineSpansets(existingSS[id], ss)
	}

	// choose an arbitrary spanset to be the "main" one. this field is deprecated
	if len(existing.SpanSets) > 0 {
		existing.SpanSet = existing.SpanSets[0]
	}
}

// combineSpansets "combines" spansets. This isn't actually possible so it just
// choose the spanset that has the highest "Matched" number as it is hopefully
// more representative of the spanset
func combineSpansets(existing *tempopb.SpanSet, new *tempopb.SpanSet) {
	if existing.Matched >= new.Matched {
		return
	}

	existing.Matched = new.Matched
	existing.Attributes = new.Attributes
	existing.Spans = new.Spans
}

func spansetID(ss *tempopb.SpanSet) string {
	id := ""

	for _, s := range ss.Attributes {
		// any attributes that start with "by" are considered to be part of the spanset identity
		if strings.HasPrefix(s.Key, "by") {
			id += s.Key + s.Value.String()
		}
	}

	return id
}

type QueryRangeCombiner struct {
	req     *tempopb.QueryRangeRequest
	e       *MetricsFrontendEvaluator
	metrics *tempopb.SearchMetrics
}

func QueryRangeCombinerFor(req *tempopb.QueryRangeRequest) *QueryRangeCombiner {
	e, _ := NewEngine().CompileMetricsQueryRangeFrontend(req)

	return &QueryRangeCombiner{
		req:     req,
		e:       e,
		metrics: &tempopb.SearchMetrics{},
	}
}

func (q *QueryRangeCombiner) Combine(resp *tempopb.QueryRangeResponse) {
	if resp == nil {
		return
	}

	// Here is where the job results are reentered into the pipeline
	q.e.ObserveJob(resp.Series)

	if resp.Metrics != nil {
		q.metrics.TotalJobs += resp.Metrics.TotalJobs
		q.metrics.TotalBlocks += resp.Metrics.TotalBlocks
		q.metrics.TotalBlockBytes += resp.Metrics.TotalBlockBytes
		q.metrics.InspectedBytes += resp.Metrics.InspectedBytes
		q.metrics.InspectedTraces += resp.Metrics.InspectedTraces
		q.metrics.InspectedSpans += resp.Metrics.InspectedSpans
		q.metrics.CompletedJobs += resp.Metrics.CompletedJobs
	}
}

func (q *QueryRangeCombiner) Response() *tempopb.QueryRangeResponse {
	return &tempopb.QueryRangeResponse{
		Series:  q.e.Results().ToProto(q.req),
		Metrics: q.metrics,
	}
}

type SeriesCombiner interface {
	Combine([]*tempopb.TimeSeries)
	Results() SeriesSet
}

type BasicCombiner struct {
	ss               SeriesSet
	len              int
	start, end, step uint64
}

func NewBasicCombiner(req *tempopb.QueryRangeRequest) *BasicCombiner {
	return &BasicCombiner{
		ss:    make(SeriesSet),
		len:   IntervalCount(req.Start, req.End, req.Step),
		start: req.Start,
		end:   req.End,
		step:  req.Step,
	}
}

func (b *BasicCombiner) Combine(in []*tempopb.TimeSeries) {
	for _, ts := range in {
		existing, ok := b.ss[ts.PromLabels]
		if !ok {
			// Convert proto labels to traceql labels
			labels := make(Labels, 0, len(ts.Labels))
			for _, l := range ts.Labels {
				labels = append(labels, Label{
					Name:  l.Key,
					Value: StaticFromAnyValue(l.Value),
				})
			}

			existing = TimeSeries{
				Labels: labels,
				Values: make([]float64, b.len),
			}
			b.ss[ts.PromLabels] = existing
		}

		for _, sample := range ts.Samples {
			ts := uint64(time.Duration(sample.TimestampMs) * time.Millisecond)
			j := IntervalOf(ts, b.start, b.end, b.step)
			if j >= 0 && j < len(existing.Values) {
				existing.Values[j] += sample.Value
			}
		}
	}
}

func (b *BasicCombiner) Results() SeriesSet {
	return b.ss
}

type hist struct {
	labels Labels
	hist   [][64]int // There is an array of powers-of-two buckets for every point in time
}

type HistogramCombiner struct {
	ss               map[string]hist
	qs               []float64
	div              float64
	len              int
	start, end, step uint64
}

func NewHistogramCombiner(req *tempopb.QueryRangeRequest, qs []float64, div float64) *HistogramCombiner {
	return &HistogramCombiner{
		div:   div,
		qs:    qs,
		ss:    make(map[string]hist),
		len:   IntervalCount(req.Start, req.End, req.Step),
		start: req.Start,
		end:   req.End,
		step:  req.Step,
	}
}

func (h *HistogramCombiner) Combine(in []*tempopb.TimeSeries) {
	for _, ts := range in {
		// Convert proto labels to traceql labels
		// while at the same time stripping the bucket label
		withoutBucket := make(Labels, 0, len(ts.Labels))
		bucket := -1
		for _, l := range ts.Labels {
			if l.Key == internalLabelBucket {
				bucket = int(l.Value.GetIntValue())
				continue
			}
			withoutBucket = append(withoutBucket, Label{
				Name:  l.Key,
				Value: StaticFromAnyValue(l.Value),
			})
		}

		if bucket < 0 || bucket > 64 {
			// Bad __bucket label?
			continue
		}

		withoutBucketStr := withoutBucket.String()

		existing, ok := h.ss[withoutBucketStr]
		if !ok {
			existing = hist{
				labels: withoutBucket,
				hist:   make([][64]int, h.len),
			}
			h.ss[withoutBucketStr] = existing
		}

		for _, sample := range ts.Samples {
			ts := uint64(time.Duration(sample.TimestampMs) * time.Millisecond)
			j := IntervalOf(ts, h.start, h.end, h.step)
			if j >= 0 && j < len(existing.hist) {
				existing.hist[j][bucket] += int(sample.Value)
			}
		}
	}
}

func (h *HistogramCombiner) Results() SeriesSet {
	results := make(SeriesSet, len(h.ss)*len(h.qs))

	for _, in := range h.ss {
		// For each input series, we create a new series for each quantile.
		for _, q := range h.qs {
			// Append label for the quantile
			labels := append((Labels)(nil), in.labels...)
			labels = append(labels, Label{"p", NewStaticFloat(q)})
			s := labels.String()

			new := TimeSeries{
				Labels: labels,
				Values: make([]float64, len(in.hist)),
			}
			for i := range in.hist {
				new.Values[i] = Percentile(q, in.hist[i]) / h.div
			}
			results[s] = new
		}
	}
	return results
}

// Percentile returns the p-value given powers-of-two bucket counts. Uses
// exponential interpolation. The original values are int64 so there are always 64 buckets.
func Percentile(p float64, buckets [64]int) float64 {
	if math.IsNaN(p) ||
		p < 0 ||
		p > 1 {
		return 0
	}

	totalCount := 0
	for _, b := range buckets {
		totalCount += b
	}

	if totalCount == 0 {
		return 0
	}

	// Maximum amount of samples to include. We round up to better handle
	// percentiles on low sample counts (<100).
	maxSamples := int(math.Ceil(p * float64(totalCount)))

	// Find the bucket where the percentile falls in
	// and the total sample count less than or equal
	// to that bucket.
	var total, bucket int
	for b, count := range buckets {
		if total+count <= maxSamples {
			bucket = b
			total += count

			if total < maxSamples {
				continue
			}
		}

		// We have enough
		break
	}

	// Fraction to interpolate between buckets, sample-count wise.
	// 0.5 means halfway
	var interp float64
	if maxSamples-total > 0 {
		interp = float64(maxSamples-total) / float64(buckets[bucket+1])
	}

	// Exponential interpolation between buckets
	minDur := math.Pow(2, float64(bucket))
	dur := minDur * math.Pow(2, interp)

	return dur
}

var (
	_ SeriesCombiner = (*BasicCombiner)(nil)
	_ SeriesCombiner = (*HistogramCombiner)(nil)
)
