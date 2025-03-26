// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxscope // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxscope"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
)

const (
	Name       = "scope"
	LegacyName = "instrumentation_scope"
	DocRef     = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlscope"
)

type Context interface {
	GetInstrumentationScope() pcommon.InstrumentationScope
	GetScopeSchemaURLItem() ctxcommon.SchemaURLItem
}
