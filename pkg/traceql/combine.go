package traceql

import (
	"slices"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

type MetadataCombiner struct {
	trs            map[string]*tempopb.TraceSearchMetadata
	trsSorted      []*tempopb.TraceSearchMetadata
	keepMostRecent int
}

func NewMetadataCombiner(keepMostRecent int) *MetadataCombiner {
	return &MetadataCombiner{
		trs:            make(map[string]*tempopb.TraceSearchMetadata, keepMostRecent),
		trsSorted:      make([]*tempopb.TraceSearchMetadata, 0, keepMostRecent),
		keepMostRecent: keepMostRecent,
	}
}

// AddSpanset adds a new spanset to the combiner. It only performs the asTraceSearchMetadata
// conversion if the spanset will be added
func (c *MetadataCombiner) AddSpanset(new *Spanset) {
	// if we're not configured to keep most recent then just add it
	if c.keepMostRecent == 0 || c.Count() < c.keepMostRecent {
		c.AddMetadata(asTraceSearchMetadata(new))
		return
	}

	// else let's see if it's worth converting this to a metadata and adding it
	// if it's already in the list, then we should add it
	if _, ok := c.trs[util.TraceIDToHexString(new.TraceID)]; ok {
		c.AddMetadata(asTraceSearchMetadata(new))
		return
	}

	// if it's within range
	if c.OldestTimestampNanos() <= new.StartTimeUnixNanos {
		c.AddMetadata(asTraceSearchMetadata(new))
		return
	}

	// this spanset is too old to bother converting and adding it
}

// AddMetadata adds the new metadata to the map. if it already exists
// use CombineSearchResults to combine the two
func (c *MetadataCombiner) AddMetadata(new *tempopb.TraceSearchMetadata) bool {
	if existing, ok := c.trs[new.TraceID]; ok {
		combineSearchResults(existing, new)
		return true
	}

	if c.Count() == c.keepMostRecent && c.keepMostRecent > 0 {
		// if this is older than the oldest element, bail
		if c.OldestTimestampNanos() > new.StartTimeUnixNano {
			return false
		}

		// otherwise remove the oldest element and we'll add the new one below
		oldest := c.trsSorted[c.Count()-1]
		delete(c.trs, oldest.TraceID)
		c.trsSorted = c.trsSorted[:len(c.trsSorted)-1]
	}

	// insert new in the right spot
	c.trs[new.TraceID] = new
	idx, _ := slices.BinarySearchFunc(c.trsSorted, new, func(a, b *tempopb.TraceSearchMetadata) int {
		if a.StartTimeUnixNano > b.StartTimeUnixNano {
			return -1
		}
		return 1
	})
	c.trsSorted = slices.Insert(c.trsSorted, idx, new)
	return true
}

func (c *MetadataCombiner) Count() int {
	return len(c.trs)
}

func (c *MetadataCombiner) Exists(id string) bool {
	_, ok := c.trs[id]
	return ok
}

func (c *MetadataCombiner) Metadata() []*tempopb.TraceSearchMetadata {
	return c.trsSorted
}

// MetadataAfter returns all traces that started after the given time
func (c *MetadataCombiner) MetadataAfter(afterSeconds uint32) []*tempopb.TraceSearchMetadata {
	afterNanos := uint64(afterSeconds) * uint64(time.Second)
	afterTraces := make([]*tempopb.TraceSearchMetadata, 0, len(c.trsSorted))

	for _, tr := range c.trsSorted {
		if tr.StartTimeUnixNano > afterNanos {
			afterTraces = append(afterTraces, tr)
		}
	}

	return afterTraces
}

func (c *MetadataCombiner) OldestTimestampNanos() uint64 {
	if len(c.trsSorted) == 0 {
		return 0
	}

	return c.trsSorted[len(c.trsSorted)-1].StartTimeUnixNano
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
	minTS, maxTS uint64
}

type QueryRangeCombiner struct {
	req     *tempopb.QueryRangeRequest
	eval    *MetricsFrontendEvaluator
	metrics *tempopb.SearchMetrics

	// used to track which series were updated since the previous diff
	// todo: it may not be worth it to track the diffs per series. it would be simpler (and possibly nearly as effective) to just calculate a global
	//  max/min for all series
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
		return tsr.minTS, tsr.maxTS, ok
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

		// Normalize into request alignment by converting timestamp into index and back
		// TimestampMs may not match exactly when we trim things around blocks, and the generators
		// This is mainly for instant queries that have large steps and few samples.
		idxMin := IntervalOfMs(series.Samples[0].TimestampMs, q.req.Start, q.req.End, q.req.Step)
		idxMax := IntervalOfMs(series.Samples[len(series.Samples)-1].TimestampMs, q.req.Start, q.req.End, q.req.Step)

		nanoMin := TimestampOf(uint64(idxMin), q.req.Start, q.req.Step)
		nanoMax := TimestampOf(uint64(idxMax), q.req.Start, q.req.Step)

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
