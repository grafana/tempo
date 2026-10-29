// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/metadata"
)

// ValidateFeatureGates returns an error if an invalid feature gate combination is configured.
// It should be called at component startup to fail fast before any data flows.
func ValidateFeatureGates() error {
	if metadata.PkgTranslatorZipkinDontEmitV0NetworkConventionsFeatureGate.IsEnabled() &&
		!metadata.PkgTranslatorZipkinEmitV1NetworkConventionsFeatureGate.IsEnabled() {
		return errors.New("pkg.translator.zipkin.DontEmitV0NetworkConventions cannot be enabled without enabling pkg.translator.zipkin.EmitV1NetworkConventions")
	}
	return nil
}
