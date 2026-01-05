package main

import (
	"github.com/go-kit/log"
)

// TraceMetrics contains metrics about the trace query
type TraceMetrics struct {
	InstancesQueried   int  `json:"instancesQueried"`
	InstancesResponded int  `json:"instancesResponded"`
	InstancesWithTrace int  `json:"instancesWithTrace"`
	InstancesNotFound  int  `json:"instancesNotFound"`
	InstancesFailed    int  `json:"instancesFailed"`
	TotalSpans         int  `json:"totalSpans"`
	PartialResponse    bool `json:"partialResponse"`
}

// CombineMetadata contains metadata about the combine operation
type CombineMetadata struct {
	InstancesQueried   int
	InstancesResponded int
	InstancesWithTrace int
	InstancesNotFound  int
	InstancesFailed    int
	TotalSpans         int
	PartialResponse    bool
	Errors             []string
}

// TraceCombiner combines traces from multiple Tempo instances
type TraceCombiner struct {
	maxSizeBytes int
	logger       log.Logger
}

// NewTraceCombiner creates a new trace combiner
func NewTraceCombiner(maxSizeBytes int, logger log.Logger) *TraceCombiner {
	return &TraceCombiner{
		maxSizeBytes: maxSizeBytes,
		logger:       logger,
	}
}
