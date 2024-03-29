// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	tagInstanceName, _ = tag.NewKey("name")
	tagPartition, _    = tag.NewKey("partition")

	statMessageCount     = stats.Int64("kafka_receiver_messages", "Number of received messages", stats.UnitDimensionless)
	statMessageOffset    = stats.Int64("kafka_receiver_current_offset", "Current message offset", stats.UnitDimensionless)
	statMessageOffsetLag = stats.Int64("kafka_receiver_offset_lag", "Current offset lag", stats.UnitDimensionless)

	statPartitionStart = stats.Int64("kafka_receiver_partition_start", "Number of started partitions", stats.UnitDimensionless)
	statPartitionClose = stats.Int64("kafka_receiver_partition_close", "Number of finished partitions", stats.UnitDimensionless)

	statUnmarshalFailedMetricPoints = stats.Int64("kafka_receiver_unmarshal_failed_metric_points", "Number of metric points failed to be unmarshaled", stats.UnitDimensionless)
	statUnmarshalFailedLogRecords   = stats.Int64("kafka_receiver_unmarshal_failed_log_records", "Number of log records failed to be unmarshaled", stats.UnitDimensionless)
	statUnmarshalFailedSpans        = stats.Int64("kafka_receiver_unmarshal_failed_spans", "Number of spans failed to be unmarshaled", stats.UnitDimensionless)
)

// metricViews return metric views for Kafka receiver.
func metricViews() []*view.View {
	partitionAgnosticTagKeys := []tag.Key{tagInstanceName}
	partitionSpecificTagKeys := []tag.Key{tagInstanceName, tagPartition}

	countMessages := &view.View{
		Name:        statMessageCount.Name(),
		Measure:     statMessageCount,
		Description: statMessageCount.Description(),
		TagKeys:     partitionSpecificTagKeys,
		Aggregation: view.Sum(),
	}

	lastValueOffset := &view.View{
		Name:        statMessageOffset.Name(),
		Measure:     statMessageOffset,
		Description: statMessageOffset.Description(),
		TagKeys:     partitionSpecificTagKeys,
		Aggregation: view.LastValue(),
	}

	lastValueOffsetLag := &view.View{
		Name:        statMessageOffsetLag.Name(),
		Measure:     statMessageOffsetLag,
		Description: statMessageOffsetLag.Description(),
		TagKeys:     partitionSpecificTagKeys,
		Aggregation: view.LastValue(),
	}

	countPartitionStart := &view.View{
		Name:        statPartitionStart.Name(),
		Measure:     statPartitionStart,
		Description: statPartitionStart.Description(),
		TagKeys:     partitionAgnosticTagKeys,
		Aggregation: view.Sum(),
	}

	countPartitionClose := &view.View{
		Name:        statPartitionClose.Name(),
		Measure:     statPartitionClose,
		Description: statPartitionClose.Description(),
		TagKeys:     partitionAgnosticTagKeys,
		Aggregation: view.Sum(),
	}

	countUnmarshalFailedMetricPoints := &view.View{
		Name:        statUnmarshalFailedMetricPoints.Name(),
		Measure:     statUnmarshalFailedMetricPoints,
		Description: statUnmarshalFailedMetricPoints.Description(),
		TagKeys:     partitionAgnosticTagKeys,
		Aggregation: view.Sum(),
	}

	countUnmarshalFailedLogRecords := &view.View{
		Name:        statUnmarshalFailedLogRecords.Name(),
		Measure:     statUnmarshalFailedLogRecords,
		Description: statUnmarshalFailedLogRecords.Description(),
		TagKeys:     partitionAgnosticTagKeys,
		Aggregation: view.Sum(),
	}

	countUnmarshalFailedSpans := &view.View{
		Name:        statUnmarshalFailedSpans.Name(),
		Measure:     statUnmarshalFailedSpans,
		Description: statUnmarshalFailedSpans.Description(),
		TagKeys:     partitionAgnosticTagKeys,
		Aggregation: view.Sum(),
	}

	return []*view.View{
		countMessages,
		lastValueOffset,
		lastValueOffsetLag,
		countPartitionStart,
		countPartitionClose,
		countUnmarshalFailedMetricPoints,
		countUnmarshalFailedLogRecords,
		countUnmarshalFailedSpans,
	}
}
