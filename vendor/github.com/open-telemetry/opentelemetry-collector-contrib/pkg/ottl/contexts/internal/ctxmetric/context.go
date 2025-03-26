// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxmetric // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"

import "go.opentelemetry.io/collector/pdata/pmetric"

const (
	Name   = "metric"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlmetric"
)

type Context interface {
	GetMetric() pmetric.Metric
}
