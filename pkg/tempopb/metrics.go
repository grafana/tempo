package tempopb

// MergeSearchMetrics adds src into dst and returns dst. If src is nil, dst is returned unchanged.
// If dst is nil and src is non-nil, dst is allocated.
func MergeSearchMetrics(dst, src *SearchMetrics) *SearchMetrics {
	if src == nil {
		return dst
	}
	if dst == nil {
		dst = &SearchMetrics{}
	}

	dst.InspectedTraces += src.InspectedTraces
	dst.InspectedBytes += src.InspectedBytes
	dst.TotalBlocks += src.TotalBlocks
	dst.CompletedJobs += src.CompletedJobs
	dst.TotalJobs += src.TotalJobs
	dst.TotalBlockBytes += src.TotalBlockBytes
	dst.InspectedSpans += src.InspectedSpans
	dst.BackendReads += src.BackendReads
	dst.BackendBytes += src.BackendBytes
	if len(src.AdditionalMetrics) > 0 {
		if dst.AdditionalMetrics == nil {
			dst.AdditionalMetrics = make(map[string]int64, len(src.AdditionalMetrics))
		}
		for k, v := range src.AdditionalMetrics {
			dst.AdditionalMetrics[k] += v
		}
	}

	return dst
}

// MergeMetadataMetrics adds src into dst and returns dst. If src is nil, dst is returned unchanged.
// If dst is nil and src is non-nil, dst is allocated.
func MergeMetadataMetrics(dst, src *MetadataMetrics) *MetadataMetrics {
	if src == nil {
		return dst
	}
	if dst == nil {
		dst = &MetadataMetrics{}
	}

	dst.InspectedBytes += src.InspectedBytes
	dst.TotalJobs += src.TotalJobs
	dst.CompletedJobs += src.CompletedJobs
	dst.TotalBlocks += src.TotalBlocks
	dst.TotalBlockBytes += src.TotalBlockBytes
	dst.BackendReads += src.BackendReads
	dst.BackendBytes += src.BackendBytes
	if len(src.AdditionalMetrics) > 0 {
		if dst.AdditionalMetrics == nil {
			dst.AdditionalMetrics = make(map[string]int64, len(src.AdditionalMetrics))
		}
		for k, v := range src.AdditionalMetrics {
			dst.AdditionalMetrics[k] += v
		}
	}

	return dst
}

// MergeTraceByIDMetrics adds src into dst and returns dst. If src is nil, dst is returned unchanged.
// If dst is nil and src is non-nil, dst is allocated.
func MergeTraceByIDMetrics(dst, src *TraceByIDMetrics) *TraceByIDMetrics {
	if src == nil {
		return dst
	}
	if dst == nil {
		dst = &TraceByIDMetrics{}
	}

	dst.InspectedBytes += src.InspectedBytes
	dst.BackendReads += src.BackendReads
	dst.BackendBytes += src.BackendBytes
	if len(src.AdditionalMetrics) > 0 {
		if dst.AdditionalMetrics == nil {
			dst.AdditionalMetrics = make(map[string]int64, len(src.AdditionalMetrics))
		}
		for k, v := range src.AdditionalMetrics {
			dst.AdditionalMetrics[k] += v
		}
	}

	return dst
}
