// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxdatapoint // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxdatapoint"

import (
	"maps"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxmetric"
)

var SymbolTable = func() map[ottl.EnumSymbol]ottl.Enum {
	st := map[ottl.EnumSymbol]ottl.Enum{
		"FLAG_NONE":              0,
		"FLAG_NO_RECORDED_VALUE": 1,
	}
	maps.Copy(st, ctxmetric.SymbolTable)
	return st
}()
