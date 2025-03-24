// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxspanevent // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxspanevent"

import "go.opentelemetry.io/collector/pdata/ptrace"

const (
	Name   = "spanevent"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlspanevent"
)

type Context interface {
	GetSpanEvent() ptrace.SpanEvent
	GetEventIndex() (int64, error)
}
