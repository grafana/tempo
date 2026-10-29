// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxexemplar // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxexemplar"

import "go.opentelemetry.io/collector/pdata/pmetric"

const (
	Name   = "exemplar"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlexemplar"
)

type Context interface {
	GetExemplar() pmetric.Exemplar
}
