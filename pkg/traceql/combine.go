package traceql

import (
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

	// Combine service stats
	// It's possible to find multiple trace fragments that satisfy a TraceQL result,
	// therefore we use max() to merge the ServiceStats.
	for service, incomingStats := range incoming.ServiceStats {
		existingStats, ok := existing.ServiceStats[service]
		if !ok {
			existingStats = &tempopb.ServiceStats{}
			if existing.ServiceStats == nil {
				existing.ServiceStats = make(map[string]*tempopb.ServiceStats)
			}
			existing.ServiceStats[service] = existingStats
		}
		existingStats.SpanCount = max(existingStats.SpanCount, incomingStats.SpanCount)
		existingStats.ErrorCount = max(existingStats.ErrorCount, incomingStats.ErrorCount)
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

type tsRange struct {
	minTS, maxTS int64
}

type QueryRangeCombiner struct {
	req     *tempopb.QueryRangeRequest
	eval    *MetricsFrontendEvaluator
	metrics *tempopb.SearchMetrics

	// used to track which series were updated since the previous diff
	//  jpe - is it worth calculating this per series? if all responses roughly return all series should we just track min/max globally?
	seriesUpdated map[string]tsRange
}

func QueryRangeCombinerFor(req *tempopb.QueryRangeRequest, mode AggregateMode, trackDiffs bool) (*QueryRangeCombiner, error) {
	eval, err := NewEngine().CompileMetricsQueryRangeNonRaw(req, mode)
	if err != nil {
		return nil, err
	}

	var seriesUpdated map[string]tsRange
	if trackDiffs {
		seriesUpdated = map[string]tsRange{}
	}

	return &QueryRangeCombiner{
		req:           req,
		eval:          eval,
		metrics:       &tempopb.SearchMetrics{},
		seriesUpdated: seriesUpdated,
	}, nil
}

func (q *QueryRangeCombiner) Combine(resp *tempopb.QueryRangeResponse) {
	if resp == nil {
		return
	}

	// mark min/max for all series
	q.markUpdatedRanges(resp)

	// Here is where the job results are reentered into the pipeline
	q.eval.ObserveSeries(resp.Series)

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
		Series:  q.eval.Results().ToProto(q.req),
		Metrics: q.metrics,
	}
}

func (q *QueryRangeCombiner) Diff() *tempopb.QueryRangeResponse {
	if q.seriesUpdated == nil {
		return q.Response()
	}

	seriesRangeFn := func(promLabels string) (uint64, uint64, bool) {
		tsr, ok := q.seriesUpdated[promLabels]
		return uint64(tsr.minTS), uint64(tsr.maxTS), ok
	}

	// filter out series that haven't change
	resp := &tempopb.QueryRangeResponse{
		Series:  q.eval.Results().ToProtoDiff(q.req, seriesRangeFn),
		Metrics: q.metrics,
	}

	// wipe out the diff for the next call
	clear(q.seriesUpdated)

	return resp
}

func (q *QueryRangeCombiner) markUpdatedRanges(resp *tempopb.QueryRangeResponse) {
	if q.seriesUpdated == nil {
		return
	}

	// mark all ranges that changed
	for _, series := range resp.Series {
		if len(series.Samples) == 0 {
			continue
		}

		nanoMin := series.Samples[0].TimestampMs * int64(time.Millisecond)
		nanoMax := series.Samples[len(series.Samples)-1].TimestampMs * int64(time.Millisecond)

		tsr, ok := q.seriesUpdated[series.PromLabels]
		if !ok {
			q.seriesUpdated[series.PromLabels] = tsRange{minTS: nanoMin, maxTS: nanoMax}
			continue
		}

		var updated bool
		if nanoMin < tsr.minTS {
			updated = true
			tsr.minTS = nanoMin
		}
		if nanoMax > tsr.maxTS {
			updated = true
			tsr.maxTS = nanoMax
		}

		if updated {
			q.seriesUpdated[series.PromLabels] = tsr
		}
	}
}
