package overrides

// Merge returns a new Overrides with the receiver as the base and other applied on top.
// A field from other is used when it is "set" (non-nil for pointers/slices/maps, non-zero
// for value types). Unset fields in other fall through to the base value.
//
// The result is a shallow copy - it shares underlying pointer, slice, and map data with
// both inputs. Callers MUST treat the returned value as read-only. Mutating any field in
// the result will mutate the base or other input.
//
// To allow explicitly overriding a field to its zero value (e.g., setting a limit to 0),
// use a pointer type for that field.
// Nil means "not set", while a pointer to zero means "explicitly set to zero".
func (base *Overrides) Merge(other *Overrides) *Overrides {
	if other == nil {
		return base
	}
	if base == nil {
		return other
	}

	result := Overrides{
		Global:           base.Global.Merge(&other.Global),
		Ingestion:        base.Ingestion.Merge(&other.Ingestion),
		Read:             base.Read.Merge(&other.Read),
		MetricsGenerator: base.MetricsGenerator.Merge(&other.MetricsGenerator),
		Forwarders:       mergeSliceShallow(base.Forwarders, other.Forwarders),
		Compaction:       base.Compaction.Merge(&other.Compaction),
		Storage:          base.Storage.Merge(&other.Storage),
		CostAttribution:  base.CostAttribution.Merge(&other.CostAttribution),
	}

	return &result
}

func (base *GlobalOverrides) Merge(other *GlobalOverrides) GlobalOverrides {
	return GlobalOverrides{
		MaxBytesPerTrace: mergePtr(base.MaxBytesPerTrace, other.MaxBytesPerTrace),
	}
}

func (base *IngestionOverrides) Merge(other *IngestionOverrides) IngestionOverrides {
	return IngestionOverrides{
		RateStrategy:           mergeVal(base.RateStrategy, other.RateStrategy),
		RateLimitBytes:         mergePtr(base.RateLimitBytes, other.RateLimitBytes),
		BurstSizeBytes:         mergePtr(base.BurstSizeBytes, other.BurstSizeBytes),
		MaxLocalTracesPerUser:  mergePtr(base.MaxLocalTracesPerUser, other.MaxLocalTracesPerUser),
		MaxGlobalTracesPerUser: mergePtr(base.MaxGlobalTracesPerUser, other.MaxGlobalTracesPerUser),
		TenantShardSize:        mergePtr(base.TenantShardSize, other.TenantShardSize),
		MaxAttributeBytes:      mergePtr(base.MaxAttributeBytes, other.MaxAttributeBytes),
		ArtificialDelay:        mergePtr(base.ArtificialDelay, other.ArtificialDelay),
		RetryInfoEnabled:       mergePtr(base.RetryInfoEnabled, other.RetryInfoEnabled),
	}
}

func (base *ReadOverrides) Merge(other *ReadOverrides) ReadOverrides {
	return ReadOverrides{
		MaxBytesPerTagValuesQuery:  mergePtr(base.MaxBytesPerTagValuesQuery, other.MaxBytesPerTagValuesQuery),
		MaxBlocksPerTagValuesQuery: mergePtr(base.MaxBlocksPerTagValuesQuery, other.MaxBlocksPerTagValuesQuery),
		MaxSearchDuration:          mergePtr(base.MaxSearchDuration, other.MaxSearchDuration),
		MaxMetricsDuration:         mergePtr(base.MaxMetricsDuration, other.MaxMetricsDuration),
		UnsafeQueryHints:           mergePtr(base.UnsafeQueryHints, other.UnsafeQueryHints),
		LeftPadTraceIDs:            mergePtr(base.LeftPadTraceIDs, other.LeftPadTraceIDs),
	}
}

func (base *CompactionOverrides) Merge(other *CompactionOverrides) CompactionOverrides {
	return CompactionOverrides{
		BlockRetention:     mergePtr(base.BlockRetention, other.BlockRetention),
		CompactionWindow:   mergePtr(base.CompactionWindow, other.CompactionWindow),
		CompactionDisabled: mergePtr(base.CompactionDisabled, other.CompactionDisabled),
	}
}

func (base *StorageOverrides) Merge(other *StorageOverrides) StorageOverrides {
	return StorageOverrides{
		DedicatedColumns: mergeSliceShallow(base.DedicatedColumns, other.DedicatedColumns),
	}
}

func (base *CostAttributionOverrides) Merge(other *CostAttributionOverrides) CostAttributionOverrides {
	return CostAttributionOverrides{
		MaxCardinality: mergePtr(base.MaxCardinality, other.MaxCardinality),
		Dimensions:     mergeMap(base.Dimensions, other.Dimensions),
	}
}

func (base *MetricsGeneratorOverrides) Merge(other *MetricsGeneratorOverrides) MetricsGeneratorOverrides {
	return MetricsGeneratorOverrides{
		RingSize:                        mergePtr(base.RingSize, other.RingSize),
		Processors:                      mergeMap(base.Processors, other.Processors),
		MaxActiveSeries:                 mergePtr(base.MaxActiveSeries, other.MaxActiveSeries),
		MaxActiveEntities:               mergePtr(base.MaxActiveEntities, other.MaxActiveEntities),
		CollectionInterval:              mergePtr(base.CollectionInterval, other.CollectionInterval),
		DisableCollection:               mergePtr(base.DisableCollection, other.DisableCollection),
		GenerateNativeHistograms:        mergeVal(base.GenerateNativeHistograms, other.GenerateNativeHistograms),
		TraceIDLabelName:                mergeVal(base.TraceIDLabelName, other.TraceIDLabelName),
		RemoteWriteHeaders:              mergeMap(base.RemoteWriteHeaders, other.RemoteWriteHeaders),
		IngestionSlack:                  mergePtr(base.IngestionSlack, other.IngestionSlack),
		NativeHistogramBucketFactor:     mergePtr(base.NativeHistogramBucketFactor, other.NativeHistogramBucketFactor),
		NativeHistogramMaxBucketNumber:  mergePtr(base.NativeHistogramMaxBucketNumber, other.NativeHistogramMaxBucketNumber),
		NativeHistogramMinResetDuration: mergePtr(base.NativeHistogramMinResetDuration, other.NativeHistogramMinResetDuration),
		SpanNameSanitization:            mergePtr(base.SpanNameSanitization, other.SpanNameSanitization),
		MaxCardinalityPerLabel:          mergePtr(base.MaxCardinalityPerLabel, other.MaxCardinalityPerLabel),
		Forwarder:                       base.Forwarder.Merge(&other.Forwarder),
		Processor:                       base.Processor.Merge(&other.Processor),
	}
}

func (base *ForwarderOverrides) Merge(other *ForwarderOverrides) ForwarderOverrides {
	return ForwarderOverrides{
		QueueSize: mergeVal(base.QueueSize, other.QueueSize),
		Workers:   mergeVal(base.Workers, other.Workers),
	}
}

func (base *ProcessorOverrides) Merge(other *ProcessorOverrides) ProcessorOverrides {
	return ProcessorOverrides{
		ServiceGraphs: base.ServiceGraphs.Merge(&other.ServiceGraphs),
		SpanMetrics:   base.SpanMetrics.Merge(&other.SpanMetrics),
		HostInfo:      base.HostInfo.Merge(&other.HostInfo),
	}
}

func (base *ServiceGraphsOverrides) Merge(other *ServiceGraphsOverrides) ServiceGraphsOverrides {
	return ServiceGraphsOverrides{
		HistogramBuckets:                      mergeSliceShallow(base.HistogramBuckets, other.HistogramBuckets),
		Dimensions:                            mergeSliceShallow(base.Dimensions, other.Dimensions),
		PeerAttributes:                        mergeSliceShallow(base.PeerAttributes, other.PeerAttributes),
		FilterPolicies:                        mergeSliceShallow(base.FilterPolicies, other.FilterPolicies),
		EnableClientServerPrefix:              mergePtr(base.EnableClientServerPrefix, other.EnableClientServerPrefix),
		EnableMessagingSystemLatencyHistogram: mergePtr(base.EnableMessagingSystemLatencyHistogram, other.EnableMessagingSystemLatencyHistogram),
		EnableVirtualNodeLabel:                mergePtr(base.EnableVirtualNodeLabel, other.EnableVirtualNodeLabel),
		SpanMultiplierKey:                     mergePtr(base.SpanMultiplierKey, other.SpanMultiplierKey),
	}
}

func (base *SpanMetricsOverrides) Merge(other *SpanMetricsOverrides) SpanMetricsOverrides {
	return SpanMetricsOverrides{
		HistogramBuckets:             mergeSliceShallow(base.HistogramBuckets, other.HistogramBuckets),
		Dimensions:                   mergeSliceShallow(base.Dimensions, other.Dimensions),
		IntrinsicDimensions:          mergeMap(base.IntrinsicDimensions, other.IntrinsicDimensions),
		FilterPolicies:               mergeSliceShallow(base.FilterPolicies, other.FilterPolicies),
		DimensionMappings:            mergeSliceShallow(base.DimensionMappings, other.DimensionMappings),
		EnableTargetInfo:             mergePtr(base.EnableTargetInfo, other.EnableTargetInfo),
		TargetInfoExcludedDimensions: mergeSliceShallow(base.TargetInfoExcludedDimensions, other.TargetInfoExcludedDimensions),
		EnableInstanceLabel:          mergePtr(base.EnableInstanceLabel, other.EnableInstanceLabel),
		SpanMultiplierKey:            mergePtr(base.SpanMultiplierKey, other.SpanMultiplierKey),
	}
}

func (base *HostInfoOverrides) Merge(other *HostInfoOverrides) HostInfoOverrides {
	return HostInfoOverrides{
		HostIdentifiers: mergeSliceShallow(base.HostIdentifiers, other.HostIdentifiers),
		MetricName:      mergeVal(base.MetricName, other.MetricName),
	}
}

// buildMergedOverrides pre-computes a merged view for each tenant entry by
// merging it with static defaults. Called once at config load/reload time.
func buildMergedOverrides(tenantLimits TenantOverrides, defaults *Overrides) TenantOverrides {
	merged := make(TenantOverrides, len(tenantLimits))
	for tenantID, tenantOv := range tenantLimits {
		merged[tenantID] = defaults.Merge(tenantOv)
	}
	return merged
}

// mergePtr returns other if non-nil, otherwise base. No copy is made.
func mergePtr[T any](base, other *T) *T {
	if other != nil {
		return other
	}
	return base
}

// mergeVal returns other if non-zero, otherwise base.
func mergeVal[T comparable](base, other T) T {
	var zero T
	if other != zero {
		return other
	}
	return base
}

// mergeSliceShallow returns other if non-nil, otherwise base. No copy is made.
func mergeSliceShallow[T any](base, other []T) []T {
	if other != nil {
		return other
	}
	return base
}

// mergeMap returns other if non-nil, otherwise base. No copy is made.
func mergeMap[K comparable, V any](base, other map[K]V) map[K]V {
	if other != nil {
		return other
	}
	return base
}
