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
func Truncate(s *pb.Span) {
	r, ok := traceutil.TruncateResource(s.Resource)
	if !ok {
		log.Debugf("span.truncate: truncated `Resource` (max %d chars): %s", traceutil.MaxResourceLen, s.Resource)
	}
	s.Resource = r

	// Error - Nothing to do
	// Optional data, Meta & Metrics can be nil
	// Soft fail on those
	for k, v := range s.Meta {
		modified := false

		if len(k) > traceutil.MaxMetaKeyLen {
			log.Debugf("span.truncate: truncating `Meta` key (max %d chars): %s", traceutil.MaxMetaKeyLen, k)
			delete(s.Meta, k)
			k = traceutil.TruncateUTF8(k, traceutil.MaxMetaKeyLen) + "..."
			modified = true
		}

		if len(v) > traceutil.MaxMetaValLen {
			v = traceutil.TruncateUTF8(v, traceutil.MaxMetaValLen) + "..."
			modified = true
		}

		if modified {
			s.Meta[k] = v
		}
	}
	for k, v := range s.Metrics {
		if len(k) > traceutil.MaxMetricsKeyLen {
			log.Debugf("span.truncate: truncating `Metrics` key (max %d chars): %s", traceutil.MaxMetricsKeyLen, k)
			delete(s.Metrics, k)
			k = traceutil.TruncateUTF8(k, traceutil.MaxMetricsKeyLen) + "..."

			s.Metrics[k] = v
		}
	}
}
