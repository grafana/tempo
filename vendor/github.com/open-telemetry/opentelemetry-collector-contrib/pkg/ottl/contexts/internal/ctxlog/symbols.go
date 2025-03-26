// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxlog"

import (
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

var SymbolTable = map[ottl.EnumSymbol]ottl.Enum{
	"SEVERITY_NUMBER_UNSPECIFIED": ottl.Enum(plog.SeverityNumberUnspecified),
	"SEVERITY_NUMBER_TRACE":       ottl.Enum(plog.SeverityNumberTrace),
	"SEVERITY_NUMBER_TRACE2":      ottl.Enum(plog.SeverityNumberTrace2),
	"SEVERITY_NUMBER_TRACE3":      ottl.Enum(plog.SeverityNumberTrace3),
	"SEVERITY_NUMBER_TRACE4":      ottl.Enum(plog.SeverityNumberTrace4),
	"SEVERITY_NUMBER_DEBUG":       ottl.Enum(plog.SeverityNumberDebug),
	"SEVERITY_NUMBER_DEBUG2":      ottl.Enum(plog.SeverityNumberDebug2),
	"SEVERITY_NUMBER_DEBUG3":      ottl.Enum(plog.SeverityNumberDebug3),
	"SEVERITY_NUMBER_DEBUG4":      ottl.Enum(plog.SeverityNumberDebug4),
	"SEVERITY_NUMBER_INFO":        ottl.Enum(plog.SeverityNumberInfo),
	"SEVERITY_NUMBER_INFO2":       ottl.Enum(plog.SeverityNumberInfo2),
	"SEVERITY_NUMBER_INFO3":       ottl.Enum(plog.SeverityNumberInfo3),
	"SEVERITY_NUMBER_INFO4":       ottl.Enum(plog.SeverityNumberInfo4),
	"SEVERITY_NUMBER_WARN":        ottl.Enum(plog.SeverityNumberWarn),
	"SEVERITY_NUMBER_WARN2":       ottl.Enum(plog.SeverityNumberWarn2),
	"SEVERITY_NUMBER_WARN3":       ottl.Enum(plog.SeverityNumberWarn3),
	"SEVERITY_NUMBER_WARN4":       ottl.Enum(plog.SeverityNumberWarn4),
	"SEVERITY_NUMBER_ERROR":       ottl.Enum(plog.SeverityNumberError),
	"SEVERITY_NUMBER_ERROR2":      ottl.Enum(plog.SeverityNumberError2),
	"SEVERITY_NUMBER_ERROR3":      ottl.Enum(plog.SeverityNumberError3),
	"SEVERITY_NUMBER_ERROR4":      ottl.Enum(plog.SeverityNumberError4),
	"SEVERITY_NUMBER_FATAL":       ottl.Enum(plog.SeverityNumberFatal),
	"SEVERITY_NUMBER_FATAL2":      ottl.Enum(plog.SeverityNumberFatal2),
	"SEVERITY_NUMBER_FATAL3":      ottl.Enum(plog.SeverityNumberFatal3),
	"SEVERITY_NUMBER_FATAL4":      ottl.Enum(plog.SeverityNumberFatal4),
}
