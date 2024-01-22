package traceql

import (
	"sort"
	"strings"

	"github.com/grafana/tempo/pkg/tempopb"
	"golang.org/x/exp/maps"
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
	ts      map[string]*tempopb.TimeSeries
	metrics *tempopb.SearchMetrics
}

func (q *QueryRangeCombiner) Combine(resp *tempopb.QueryRangeResponse) {
	if resp == nil || len(resp.Series) == 0 {
		return
	}

	if q.ts == nil {
		q.ts = make(map[string]*tempopb.TimeSeries, len(resp.Series))
	}

	for _, series := range resp.Series {

		existing, ok := q.ts[series.PromLabels]
		if !ok {
			q.ts[series.PromLabels] = series
			continue
		}

		q.combine(series, existing)
	}

	if q.metrics == nil {
		q.metrics = &tempopb.SearchMetrics{}
	}

	if resp.Metrics != nil {
		q.metrics.TotalBlocks += resp.Metrics.TotalBlocks
		q.metrics.TotalBlockBytes += resp.Metrics.TotalBlockBytes
		q.metrics.InspectedBytes += resp.Metrics.InspectedBytes
	}
}

func (QueryRangeCombiner) combine(in *tempopb.TimeSeries, out *tempopb.TimeSeries) {
outer:
	for _, sample := range in.Samples {
		for i, existing := range out.Samples {
			if sample.TimestampMs == existing.TimestampMs {
				out.Samples[i].Value += sample.Value
				continue outer
			}
		}

		out.Samples = append(out.Samples, sample)
	}
}

func (q *QueryRangeCombiner) Response() *tempopb.QueryRangeResponse {
	if q.metrics == nil {
		q.metrics = &tempopb.SearchMetrics{}
	}
	return &tempopb.QueryRangeResponse{
		Series:  maps.Values(q.ts),
		Metrics: q.metrics,
	}
}
