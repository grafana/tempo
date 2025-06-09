package combiner

import (
	"github.com/grafana/tempo/pkg/tempopb"
)

// These structs combine response metrics in a single place

type SearchMetricsCombiner struct {
	Metrics *tempopb.SearchMetrics
}

func NewSearchMetricsCombiner() *SearchMetricsCombiner {
	return &SearchMetricsCombiner{
		Metrics: &tempopb.SearchMetrics{},
	}
}

func (mc *SearchMetricsCombiner) Combine(newMetrics *tempopb.SearchMetrics, resp PipelineResponse) {
	if newMetrics != nil {
		mc.Metrics.CompletedJobs++
		if !IsCacheHit(resp.HTTPResponse()) {
			mc.Metrics.InspectedTraces += newMetrics.InspectedTraces
			mc.Metrics.InspectedBytes += newMetrics.InspectedBytes
		}
	}
}

func (mc *SearchMetricsCombiner) CombineMetadata(newMetrics *tempopb.SearchMetrics, _ PipelineResponse) {
	// These "Total" metrics are calculated by the frontend in the sharder.
	// TotalBlockBytes is the total bytes of all blocks considered for the search, irrelevant of the cache.
	// InspectedBytes is the total bytes actually read in the Parquet files, calculated by the queriers and conditional to the response being cached.
	if newMetrics != nil {
		mc.Metrics.TotalBlocks += newMetrics.TotalBlocks
		mc.Metrics.TotalJobs += newMetrics.TotalJobs
		mc.Metrics.TotalBlockBytes += newMetrics.TotalBlockBytes
	}
}

type TraceByIDMetricsCombiner struct {
	Metrics *tempopb.TraceByIDMetrics
}

func NewTraceByIDMetricsCombiner() *TraceByIDMetricsCombiner {
	return &TraceByIDMetricsCombiner{
		Metrics: &tempopb.TraceByIDMetrics{},
	}
}

func (mc *TraceByIDMetricsCombiner) Combine(newMetrics *tempopb.TraceByIDMetrics, resp PipelineResponse) {
	if newMetrics != nil && !IsCacheHit(resp.HTTPResponse()) {
		mc.Metrics.InspectedBytes += newMetrics.InspectedBytes
	}
}

type MetadataMetricsCombiner struct {
	Metrics *tempopb.MetadataMetrics
}

func NewMetadataMetricsCombiner() *MetadataMetricsCombiner {
	return &MetadataMetricsCombiner{
		Metrics: &tempopb.MetadataMetrics{},
	}
}

func (mc *MetadataMetricsCombiner) Combine(newMetrics *tempopb.MetadataMetrics, resp PipelineResponse) {
	if newMetrics != nil && !IsCacheHit(resp.HTTPResponse()) {
		mc.Metrics.InspectedBytes += newMetrics.InspectedBytes
	}
}

type QueryRangeMetricsCombiner struct {
	Metrics *tempopb.SearchMetrics
}

func NewQueryRangeMetricsCombiner() *QueryRangeMetricsCombiner {
	return &QueryRangeMetricsCombiner{
		Metrics: &tempopb.SearchMetrics{},
	}
}

func (mc *QueryRangeMetricsCombiner) Combine(newMetrics *tempopb.SearchMetrics, resp PipelineResponse) {
	if newMetrics != nil {
		// this is a coordination between the sharder and combiner. the sharder returns one response with summary metrics
		// only. the combiner correctly takes and accumulates that job. however, if the response has no jobs this is
		// an indicator this is a "real" response so we set CompletedJobs to 1 to increment in the combiner.
		if newMetrics.TotalJobs == 0 {
			newMetrics.CompletedJobs = 1
			mc.Metrics.CompletedJobs += newMetrics.CompletedJobs
		}
		if !IsCacheHit(resp.HTTPResponse()) {
			mc.Metrics.TotalJobs += newMetrics.TotalJobs
			mc.Metrics.TotalBlocks += newMetrics.TotalBlocks
			mc.Metrics.TotalBlockBytes += newMetrics.TotalBlockBytes
			mc.Metrics.InspectedBytes += newMetrics.InspectedBytes
			mc.Metrics.InspectedTraces += newMetrics.InspectedTraces
			mc.Metrics.InspectedSpans += newMetrics.InspectedSpans
		}
	}
}
