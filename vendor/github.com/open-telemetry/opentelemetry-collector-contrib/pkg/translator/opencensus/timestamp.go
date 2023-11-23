// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensus // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// timestampAsTimestampPb converts a pcommon.Timestamp to a protobuf known type Timestamp.
func timestampAsTimestampPb(ts pcommon.Timestamp) *timestamppb.Timestamp {
	if ts == 0 {
		return nil
	}
	return timestamppb.New(ts.AsTime())
}
