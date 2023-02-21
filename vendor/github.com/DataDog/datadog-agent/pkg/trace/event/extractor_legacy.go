// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package event

import (
	"strings"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/trace/traceutil"
)

// legacyExtractor is an event extractor that decides whether to extract APM events from spans based on
// `serviceName => sampling rate` mappings.
type legacyExtractor struct {
	rateByService map[string]float64
}

// NewLegacyExtractor returns an APM event extractor that decides whether to extract APM events from spans following the
// specified extraction rates for a span's service.
func NewLegacyExtractor(rateByService map[string]float64) Extractor {
	// lower-case keys for case insensitive matching (see #3113)
	rbs := make(map[string]float64, len(rateByService))
	for k, v := range rateByService {
		rbs[strings.ToLower(k)] = v
	}
	return &legacyExtractor{rateByService: rbs}
}

// Extract decides to extract an apm event from the provided span if there's an extraction rate configured for that
// span's service. In this case the extracted event is returned along with the found extraction rate and a true value.
// If this rate doesn't exist or the provided span is not a top level one, then no extraction is done and false is
// returned as the third value, with the others being invalid.
func (e *legacyExtractor) Extract(s *pb.Span, priority sampler.SamplingPriority) (float64, bool) {
	if !traceutil.HasTopLevel(s) {
		return 0, false
	}
	extractionRate, ok := e.rateByService[strings.ToLower(s.Service)]
	if !ok {
		return 0, false
	}
	return extractionRate, true
}
