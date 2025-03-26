// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package ctxcommon // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/ctxcommon"

//revive:disable:var-naming The methods in this interface are defined by pdata types.
type SchemaURLItem interface {
	SchemaUrl() string
	SetSchemaUrl(v string)
}

//revive:enable:var-naming
