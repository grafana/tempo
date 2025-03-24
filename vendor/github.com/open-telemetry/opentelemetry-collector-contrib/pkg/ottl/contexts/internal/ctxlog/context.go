// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxlog"

import "go.opentelemetry.io/collector/pdata/plog"

const (
	Name   = "log"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottllog"
)

type Context interface {
	GetLogRecord() plog.LogRecord
}
