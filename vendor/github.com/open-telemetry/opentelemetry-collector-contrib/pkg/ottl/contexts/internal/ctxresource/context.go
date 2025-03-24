// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctxresource // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxresource"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"
)

const (
	Name   = "resource"
	DocRef = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlresource"
)

type Context interface {
	GetResource() pcommon.Resource
	GetResourceSchemaURLItem() ctxcommon.SchemaURLItem
}
