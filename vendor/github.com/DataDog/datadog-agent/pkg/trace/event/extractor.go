// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package event

import (
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
)

// Extractor extracts APM events from matching spans.
type Extractor interface {
	// Extract decides whether to extract an APM event from the provided span with the specified priority and returns
	// a suggested extraction sample rate and a bool value. If no event was extracted the bool value will be false and
	// the rate should not be used.
	Extract(span *pb.Span, priority sampler.SamplingPriority) (rate float64, ok bool)
}
