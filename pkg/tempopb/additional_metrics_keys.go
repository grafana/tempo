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
	AdditionalMetricEngineBytes        = "engineBytes"
)

// isCacheableAdditionalMetric reports whether an AdditionalMetrics key should be retained
// across query-result cache hits. Keys absent from the map are not cacheable.
var isCacheableAdditionalMetric = map[string]bool{
	AdditionalMetricEngineBytes: true,
}

// IsCacheableAdditionalMetric reports whether an AdditionalMetrics key should be retained
// across query-result cache hits.
func IsCacheableAdditionalMetric(key string) bool {
	return isCacheableAdditionalMetric[key]
}

// CacheableAdditionalMetrics returns a new map with only keys marked cacheable by
// IsCacheableAdditionalMetric. Returns nil when src has no cacheable entries.
func CacheableAdditionalMetrics(src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return nil
	}
	var dst map[string]int64
	for k, v := range src {
		if !IsCacheableAdditionalMetric(k) {
			continue
		}
		if dst == nil {
			dst = make(map[string]int64)
		}
		dst[k] = v
	}
	return dst
}
