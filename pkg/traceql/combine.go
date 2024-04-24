package traceql

import (
	"math"
	"sort"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/prometheus/model/labels"
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
	series := SeriesSetFromProto(q.req, resp.Series)
	q.e.ObserveJob(series)

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
	Combine(SeriesSet)
	Results() SeriesSet
}

type BasicCombiner struct {
	ss SeriesSet
}

func (b *BasicCombiner) Combine(in SeriesSet) {
	if b.ss == nil {
		b.ss = make(SeriesSet, len(in))
	}

	for k, ts := range in {

		existing, ok := b.ss[k]
		if !ok {
			b.ss[k] = ts
			continue
		}

		b.combine(ts, existing)
	}
}

func (BasicCombiner) combine(in TimeSeries, out TimeSeries) {
	for i := range in.Values {
		out.Values[i] += in.Values[i]
	}
}

func (b *BasicCombiner) Results() SeriesSet {
	return b.ss
}

type HistogramCombiner struct {
	// ss map[string][][64]int
	b   BasicCombiner
	qs  []float64
	by  []Attribute
	div float64
}

func (h *HistogramCombiner) Combine(in SeriesSet) {
	h.b.Combine(in)
}

func (h *HistogramCombiner) Results() SeriesSet {
	rawResults := h.b.Results()

	// Here is where we compute the final percentiles
	allBuckets := make(map[FastValues][][64]int)
	for _, rawSeries := range rawResults {
		withoutBucket := FastValues{}
		bucket := -1
		for i, l := range rawSeries.Labels {
			// TODO - It's roundtripping the internal bucket weird
			if l.Name == ".__bucket" {
				bucket = l.Value.N
				break
			}
			withoutBucket[i] = l.Value
		}

		if bucket < 0 || bucket > 64 {
			// Bad __bucket label?
			continue
		}

		existing, ok := allBuckets[withoutBucket]
		if !ok {
			existing = make([][64]int, len(rawSeries.Values))
			allBuckets[withoutBucket] = existing
		}

		for i, v := range rawSeries.Values {
			existing[i][bucket] += int(v)
		}
	}

	// Now generate new series
	outresults := make(SeriesSet)
	for labels, summed := range allBuckets {
		for _, q := range h.qs {
			l, prom := h.labelsFor(labels, q)

			new := TimeSeries{
				Labels: l,
				Values: make([]float64, len(summed)),
			}
			for i := range summed {
				new.Values[i] = percentile(q, summed[i]) / h.div
			}
			outresults[prom] = new
		}
	}
	return outresults
	// return h.b.Results()
}

func (h *HistogramCombiner) labelsFor(vals FastValues, percentile float64) ([]Label, string) {
	tempoLabels := make([]Label, 0, len(h.by))
	for i, v := range vals {
		if v.Type == TypeNil {
			continue
		}
		tempoLabels = append(tempoLabels, Label{h.by[i].String(), v})
	}
	tempoLabels = append(tempoLabels, Label{"p", NewStaticFloat(percentile)})

	// Prometheus-style version for convenience
	promLabels := labels.NewBuilder(nil)
	for _, l := range tempoLabels {
		promValue := l.Value.EncodeToString(false)
		if promValue == "" {
			promValue = "<empty>"
		}
		promLabels.Set(l.Name, promValue)
	}
	// When all nil then force one.
	if promLabels.Labels().IsEmpty() {
		promLabels.Set(h.by[0].String(), "<nil>")
	}

	return tempoLabels, promLabels.Labels().String()
}

// Percentile returns the estimated latency percentile in nanoseconds.
func percentile(p float64, buckets [64]int) float64 {
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
