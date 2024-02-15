// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	mLastConnectionState = stats.Int64("jaegerexporter_conn_state", "Last connection state: 0 = Idle, 1 = Connecting, 2 = Ready, 3 = TransientFailure, 4 = Shutdown", stats.UnitDimensionless)
	vLastConnectionState = &view.View{
		Name:        mLastConnectionState.Name(),
		Measure:     mLastConnectionState,
		Description: mLastConnectionState.Description(),
		Aggregation: view.LastValue(),
		TagKeys: []tag.Key{
			tag.MustNewKey("exporter_name"),
		},
	}
)

// metricViews return the metrics views according to given telemetry level.
func metricViews() []*view.View {
	return []*view.View{vLastConnectionState}
}
