// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinv2 // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"

import (
	"fmt"

	"go.opentelemetry.io/collector/featuregate"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/internal/metadata"
)

// ValidateFeatureGates returns an error if an invalid feature gate combination is configured.
// It should be called at component startup to fail fast before any data flows.
//
// For each semconv migration, enabling the DontEmitV0* gate without also enabling the
// matching EmitV1* gate would emit neither the deprecated nor the new attributes, silently
// dropping them; that combination is rejected here.
func ValidateFeatureGates() error {
	for _, pair := range [][2]*featuregate.Gate{
		{metadata.PkgTranslatorZipkinDontEmitV0HTTPConventionsFeatureGate, metadata.PkgTranslatorZipkinEmitV1HTTPConventionsFeatureGate},
		{metadata.PkgTranslatorZipkinDontEmitV0ScopeConventionsFeatureGate, metadata.PkgTranslatorZipkinEmitV1ScopeConventionsFeatureGate},
		{metadata.PkgTranslatorZipkinDontEmitV0CloudResourceConventionsFeatureGate, metadata.PkgTranslatorZipkinEmitV1CloudResourceConventionsFeatureGate},
	} {
		dontEmitV0, emitV1 := pair[0], pair[1]
		if dontEmitV0.IsEnabled() && !emitV1.IsEnabled() {
			return fmt.Errorf("%s cannot be enabled without enabling %s", dontEmitV0.ID(), emitV1.ID())
		}
	}
	return nil
}
