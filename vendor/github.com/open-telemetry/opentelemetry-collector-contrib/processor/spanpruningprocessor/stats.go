// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// aggregationData tracks statistics and time ranges for a group of spans in
// a single pass, replacing separate calculations for efficiency.
type aggregationData struct {
	count         int64
	minDuration   time.Duration
	maxDuration   time.Duration
	sumDuration   time.Duration
	bucketCounts  []int64
	earliestStart pcommon.Timestamp
	latestEnd     pcommon.Timestamp
}

// calculateAggregationData derives span counts and duration stats for the
// provided nodes in one traversal.
func (p *spanPruningProcessor) calculateAggregationData(nodes []*spanNode) aggregationData {
	data := aggregationData{
		count: int64(len(nodes)),
	}

	if len(p.config.AggregationHistogramBuckets) > 0 {
		data.bucketCounts = make([]int64, len(p.config.AggregationHistogramBuckets)+1)
	}

	for i, node := range nodes {
		span := node.span
		data.updateWithSpan(span, i == 0, p.config.AggregationHistogramBuckets)
	}

	return data
}

// updateWithSpan incorporates a single span into the aggregation statistics,
// tracking min/max durations and time ranges.
func (data *aggregationData) updateWithSpan(span ptrace.Span, isFirst bool, histogramBuckets []time.Duration) {
	startTime := span.StartTimestamp().AsTime()
	endTime := span.EndTimestamp().AsTime()
	duration := endTime.Sub(startTime)

	// Calculate duration statistics
	if isFirst {
		data.minDuration = duration
		data.maxDuration = duration
		data.earliestStart = span.StartTimestamp()
		data.latestEnd = span.EndTimestamp()
	} else {
		if duration < data.minDuration {
			data.minDuration = duration
		}
		if duration > data.maxDuration {
			data.maxDuration = duration
		}
		if span.StartTimestamp() < data.earliestStart {
			data.earliestStart = span.StartTimestamp()
		}
		if span.EndTimestamp() > data.latestEnd {
			data.latestEnd = span.EndTimestamp()
		}
	}
	data.sumDuration += duration

	if len(histogramBuckets) > 0 {
		bucketIndex := len(histogramBuckets)
		for i, bucket := range histogramBuckets {
			if duration <= bucket {
				bucketIndex = i
				break
			}
		}

		for i := bucketIndex; i < len(data.bucketCounts); i++ {
			data.bucketCounts[i]++
		}
	}
}
