// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package stats

import (
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

// keySamplingRateGlobal is a metric key holding the global sampling rate.
const keySamplingRateGlobal = "_sample_rate"

// weight returns the weight of the span as defined for sampling, i.e. the
// inverse of the sampling rate.
func weight(s *pb.Span) float64 {
	if s == nil {
		return 1
	}
	sampleRate, ok := s.Metrics[keySamplingRateGlobal]
	if !ok || sampleRate <= 0.0 || sampleRate > 1.0 {
		return 1
	}
	return 1.0 / sampleRate
}
