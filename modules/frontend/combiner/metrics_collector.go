package combiner

import (
	"sync/atomic"

	"github.com/grafana/tempo/pkg/tempopb"
)

// MetricsCollector provides a unified way to track metrics across different endpoints
type MetricsCollector interface {
	// Add increments the inspected bytes count
	Add(bytesProcessed uint64)
	// TotalBytes returns the total bytes processed
	TotalBytes() uint64
	// AttachTo attaches the collected metrics to the response
	AttachTo(response any)
}

// BaseMetricsCollector implements the core functionality for all collectors
type BaseMetricsCollector struct {
	inspectedBytes atomic.Uint64
}

func NewBaseMetricsCollector() *BaseMetricsCollector {
	return &BaseMetricsCollector{}
}

func (c *BaseMetricsCollector) Add(bytesProcessed uint64) {
	if bytesProcessed > 0 {
		c.inspectedBytes.Add(bytesProcessed)
	}
}

func (c *BaseMetricsCollector) TotalBytes() uint64 {
	return c.inspectedBytes.Load()
}

// TraceByIDMetricsCollector handles metrics for TraceByID endpoint
type TraceByIDMetricsCollector struct {
	*BaseMetricsCollector
}

func NewTraceByIDMetricsCollector() *TraceByIDMetricsCollector {
	return &TraceByIDMetricsCollector{NewBaseMetricsCollector()}
}

func (c *TraceByIDMetricsCollector) AttachTo(response any) {
	if resp, ok := response.(*tempopb.TraceByIDResponse); ok {
		if resp.Metrics == nil {
			resp.Metrics = &tempopb.TraceByIDMetrics{}
		}
		resp.Metrics.InspectedBytes = c.TotalBytes()
	}
}

// SearchMetricsCollector handles metrics for Search endpoint
type SearchMetricsCollector struct {
	*BaseMetricsCollector
}

func NewSearchMetricsCollector() *SearchMetricsCollector {
	return &SearchMetricsCollector{NewBaseMetricsCollector()}
}

func (c *SearchMetricsCollector) AttachTo(response any) {
	if resp, ok := response.(*tempopb.SearchResponse); ok {
		if resp.Metrics == nil {
			resp.Metrics = &tempopb.SearchMetrics{}
		}
		resp.Metrics.InspectedBytes = c.TotalBytes()
	}
}

// TagValuesMetricsCollector handles metrics for Tag Values endpoint
type TagValuesMetricsCollector struct {
	*BaseMetricsCollector
}

func NewTagValuesMetricsCollector() *TagValuesMetricsCollector {
	return &TagValuesMetricsCollector{NewBaseMetricsCollector()}
}

func (c *TagValuesMetricsCollector) AttachTo(response any) {
	switch resp := response.(type) {
	case *tempopb.SearchTagValuesResponse:
		if resp.Metrics == nil {
			resp.Metrics = &tempopb.MetadataMetrics{}
		}
		resp.Metrics.InspectedBytes = c.TotalBytes()
	case *tempopb.SearchTagValuesV2Response:
		if resp.Metrics == nil {
			resp.Metrics = &tempopb.MetadataMetrics{}
		}
		resp.Metrics.InspectedBytes = c.TotalBytes()
	}
}

// MetricsQueryRangeCollector handles metrics for Metrics Query Range endpoint
type MetricsQueryRangeCollector struct {
	*BaseMetricsCollector
}

func NewMetricsQueryRangeCollector() *MetricsQueryRangeCollector {
	return &MetricsQueryRangeCollector{NewBaseMetricsCollector()}
}

func (c *MetricsQueryRangeCollector) AttachTo(response any) {
	if resp, ok := response.(*tempopb.QueryRangeResponse); ok {
		if resp.Metrics == nil {
			resp.Metrics = &tempopb.SearchMetrics{}
		}
		resp.Metrics.InspectedBytes = c.TotalBytes()
	}
}
