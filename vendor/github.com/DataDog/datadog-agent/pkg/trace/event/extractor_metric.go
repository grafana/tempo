// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package event

import (
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
)

// metricBasedExtractor is an event extractor that decides whether to extract APM events from spans based on
// the value of the event extraction rate metric set on those spans.
type metricBasedExtractor struct{}

// NewMetricBasedExtractor returns an APM event extractor that decides whether to extract APM events from spans based on
// the value of the event extraction rate metric set on those span.
func NewMetricBasedExtractor() Extractor {
	return &metricBasedExtractor{}
}

// Extract decides whether to extract APM events from a span based on the value of the event extraction rate metric set
// on that span. If such a value exists, the extracted event is returned along with this rate and a true value.
// Otherwise, false is returned as the third value and the others are invalid.
//
// NOTE: If priority is UserKeep (manually sampled) any extraction rate bigger than 0 is upscaled to 1 to ensure no
// extraction sampling is done on this event.
func (e *metricBasedExtractor) Extract(s *pb.Span, priority sampler.SamplingPriority) (float64, bool) {
	if len(s.Metrics) == 0 {
		// metric not set
		return 0, false
	}
	extractionRate, ok := s.Metrics[sampler.KeySamplingRateEventExtraction]
	if !ok {
		return 0, false
	}
	if extractionRate > 0 && priority >= sampler.PriorityUserKeep {
		// If the trace has been manually sampled, we keep all matching spans
		extractionRate = 1
	}
	return extractionRate, true
}
