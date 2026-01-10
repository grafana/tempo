package combiner

import (
	"fmt"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CombineSearchResults combines search results from multiple instances
// Deduplicates traces by traceID and merges metrics
func (c *Combiner) CombineSearchResults(results []SearchResult) (*tempopb.SearchResponse, *SearchMetadata, error) {
	metadata := &SearchMetadata{
		InstancesQueried: len(results),
	}

	// Map to deduplicate traces by traceID
	traceMap := make(map[string]*tempopb.TraceSearchMetadata)

	// Combined metrics
	combinedMetrics := &tempopb.SearchMetrics{}

	for _, result := range results {
		if result.Error != nil {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: %v", result.Instance, result.Error))
			level.Warn(c.logger).Log("msg", "instance search failed", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.NotFound {
			metadata.InstancesFailed++
			metadata.Errors = append(metadata.Errors, fmt.Sprintf("%s: not found", result.Instance))
			level.Warn(c.logger).Log("msg", "instance returned not found", "instance", result.Instance)
			continue
		}

		metadata.InstancesResponded++

		searchResp := result.Response
		if searchResp == nil {
			continue
		}

		// Merge metrics
		if searchResp.Metrics != nil {
			combinedMetrics.InspectedTraces += searchResp.Metrics.InspectedTraces
			combinedMetrics.InspectedBytes += searchResp.Metrics.InspectedBytes
			combinedMetrics.TotalBlocks += searchResp.Metrics.TotalBlocks
			combinedMetrics.CompletedJobs += searchResp.Metrics.CompletedJobs
			combinedMetrics.TotalJobs += searchResp.Metrics.TotalJobs
			combinedMetrics.TotalBlockBytes += searchResp.Metrics.TotalBlockBytes
			combinedMetrics.InspectedSpans += searchResp.Metrics.InspectedSpans
		}

		// Deduplicate and merge traces
		for _, tr := range searchResp.Traces {
			if tr == nil {
				continue
			}

			existing, found := traceMap[tr.TraceID]
			if !found {
				// Make a copy to avoid modifying the original
				traceMap[tr.TraceID] = tr
			} else {
				// Merge trace metadata using similar logic to Tempo's combineSearchResults
				combineSearchResultMetadata(existing, tr)
			}
		}

		level.Debug(c.logger).Log("msg", "processed search results from instance", "instance", result.Instance, "traces", len(searchResp.Traces))
	}

	// Convert map to slice and sort by start time (most recent first)
	traces := make([]*tempopb.TraceSearchMetadata, 0, len(traceMap))
	for _, tr := range traceMap {
		traces = append(traces, tr)
	}

	// Sort by start time descending (most recent first)
	sortTracesByStartTime(traces)

	return &tempopb.SearchResponse{
		Traces:  traces,
		Metrics: combinedMetrics,
	}, metadata, nil
}

// combineSearchResultMetadata merges two trace search metadata entries
// This follows the same logic as Tempo's combineSearchResults in pkg/traceql/combine.go
func combineSearchResultMetadata(existing, incoming *tempopb.TraceSearchMetadata) {
	if existing.TraceID == "" {
		existing.TraceID = incoming.TraceID
	}

	if existing.RootServiceName == "" {
		existing.RootServiceName = incoming.RootServiceName
	}

	if existing.RootTraceName == "" {
		existing.RootTraceName = incoming.RootTraceName
	}

	// Earliest start time
	if existing.StartTimeUnixNano > incoming.StartTimeUnixNano || existing.StartTimeUnixNano == 0 {
		existing.StartTimeUnixNano = incoming.StartTimeUnixNano
	}

	// Longest duration
	if existing.DurationMs < incoming.DurationMs || existing.DurationMs == 0 {
		existing.DurationMs = incoming.DurationMs
	}

	// Combine service stats using max()
	for service, incomingStats := range incoming.ServiceStats {
		existingStats, ok := existing.ServiceStats[service]
		if !ok {
			existingStats = &tempopb.ServiceStats{}
			if existing.ServiceStats == nil {
				existing.ServiceStats = make(map[string]*tempopb.ServiceStats)
			}
			existing.ServiceStats[service] = existingStats
		}
		if incomingStats.SpanCount > existingStats.SpanCount {
			existingStats.SpanCount = incomingStats.SpanCount
		}
		if incomingStats.ErrorCount > existingStats.ErrorCount {
			existingStats.ErrorCount = incomingStats.ErrorCount
		}
	}

	// Combine spansets - add any new ones
	existingSS := make(map[string]bool)
	for _, ss := range existing.SpanSets {
		existingSS[spansetKey(ss)] = true
	}

	for _, ss := range incoming.SpanSets {
		key := spansetKey(ss)
		if !existingSS[key] {
			existing.SpanSets = append(existing.SpanSets, ss)
			existingSS[key] = true
		}
	}

	// Update the deprecated SpanSet field
	if len(existing.SpanSets) > 0 {
		existing.SpanSet = existing.SpanSets[0]
	}
}

// spansetKey generates a unique key for a spanset for deduplication
func spansetKey(ss *tempopb.SpanSet) string {
	if ss == nil {
		return ""
	}
	// Use the first span ID as part of the key if available
	if len(ss.Spans) > 0 && ss.Spans[0] != nil {
		return fmt.Sprintf("%x", ss.Spans[0].SpanID)
	}
	return fmt.Sprintf("matched:%d", ss.Matched)
}

// sortTracesByStartTime sorts traces by start time in descending order (most recent first)
func sortTracesByStartTime(traces []*tempopb.TraceSearchMetadata) {
	for i := 0; i < len(traces)-1; i++ {
		for j := i + 1; j < len(traces); j++ {
			if traces[i].StartTimeUnixNano < traces[j].StartTimeUnixNano {
				traces[i], traces[j] = traces[j], traces[i]
			}
		}
	}
}
