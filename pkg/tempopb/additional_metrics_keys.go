package tempopb

// AdditionalMetric* are the stable string keys used in
// SearchMetrics.AdditionalMetrics, TraceByIDMetrics.AdditionalMetrics, and
// MetadataMetrics.AdditionalMetrics. They are part of the wire contract:
// rename only with a deprecation cycle.
//
// Key naming follows lowerCamelCase to match the existing JSON shape produced
// by tempo.pb.go for related fields (e.g. "inspectedBytes", "totalJobs").
const (
	AdditionalMetricRowGroupsInspected = "rowGroupsInspected"
	AdditionalMetricRowGroupsSkipped   = "rowGroupsSkipped"
	AdditionalMetricPagesInspected     = "pagesInspected"
	AdditionalMetricPagesSkipped       = "pagesSkipped"
	AdditionalMetricCacheHits          = "cacheHits"
	AdditionalMetricCacheMisses        = "cacheMisses"
	AdditionalMetricCacheBytes         = "cacheBytes"

	// AdditionalMetricReturnedBytes is the proto Size() of the final,
	// deduped trace-by-id response actually returned to the caller. Unlike
	// the other keys in this file, it is computed once per request at the
	// point the final result is assembled, and is NOT additive across
	// per-job/per-shard responses.
	AdditionalMetricReturnedBytes = "returnedBytes"
)
