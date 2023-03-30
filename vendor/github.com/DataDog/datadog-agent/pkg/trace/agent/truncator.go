// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package agent

import (
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/traceutil"
)

// Truncate checks that the span resource, meta and metrics are within the max length
// and modifies them if they are not
func (a *Agent) Truncate(s *pb.Span) {
	r, ok := a.TruncateResource(s.Resource)
	if !ok {
		log.Debugf("span.truncate: truncated `Resource` (max %d chars): %s", a.conf.MaxResourceLen, s.Resource)
	}
	s.Resource = r

	// Error - Nothing to do
	// Optional data, Meta & Metrics can be nil
	// Soft fail on those
	for k, v := range s.Meta {
		modified := false

		if len(k) > MaxMetaKeyLen {
			log.Debugf("span.truncate: truncating `Meta` key (max %d chars): %s", MaxMetaKeyLen, k)
			delete(s.Meta, k)
			k = traceutil.TruncateUTF8(k, MaxMetaKeyLen) + "..."
			modified = true
		}

		if len(v) > MaxMetaValLen {
			v = traceutil.TruncateUTF8(v, MaxMetaValLen) + "..."
			modified = true
		}

		if modified {
			s.Meta[k] = v
		}
	}
	for k, v := range s.Metrics {
		if len(k) > MaxMetricsKeyLen {
			log.Debugf("span.truncate: truncating `Metrics` key (max %d chars): %s", MaxMetricsKeyLen, k)
			delete(s.Metrics, k)
			k = traceutil.TruncateUTF8(k, MaxMetricsKeyLen) + "..."

			s.Metrics[k] = v
		}
	}
}

const (
	// MaxMetaKeyLen the maximum length of metadata key
	MaxMetaKeyLen = 200
	// MaxMetaValLen the maximum length of metadata value
	MaxMetaValLen = 25000
	// MaxMetricsKeyLen the maximum length of a metric name key
	MaxMetricsKeyLen = MaxMetaKeyLen
)

// TruncateResource truncates a span's resource to the maximum allowed length.
// It returns true if the input was below the max size.
func (a *Agent) TruncateResource(r string) (string, bool) {
	return traceutil.TruncateUTF8(r, a.conf.MaxResourceLen), len(r) <= a.conf.MaxResourceLen
}
