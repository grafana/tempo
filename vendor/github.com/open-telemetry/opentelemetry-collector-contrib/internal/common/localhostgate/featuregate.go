// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// package localhostgate defines a feature gate that controls whether server-like receivers and extensions use localhost as the default host for their endpoints.
// This package is duplicated across core and contrib to avoid exposing the feature gate as part of the public API.
// To do this we define a `registerOrLoad` helper and try to register the gate in both modules.
// IMPORTANT NOTE: ANY CHANGES TO THIS PACKAGE MUST BE MIRRORED IN THE CORE COUNTERPART.
// See https://github.com/open-telemetry/opentelemetry-collector/blob/main/internal/localhostgate/featuregate.go
package localhostgate // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/localhostgate"

import (
	"errors"

	"go.opentelemetry.io/collector/featuregate"
)

const UseLocalHostAsDefaultHostID = "component.UseLocalHostAsDefaultHost"

// UseLocalHostAsDefaultHostfeatureGate is the feature gate that controls whether
// server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints.
var UseLocalHostAsDefaultHostfeatureGate = mustRegisterOrLoad(
	featuregate.GlobalRegistry(),
	UseLocalHostAsDefaultHostID,
	featuregate.StageStable,
	featuregate.WithRegisterToVersion("v0.111.0"),
	featuregate.WithRegisterDescription("controls whether server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints"),
)

// mustRegisterOrLoad tries to register the feature gate and loads it if it already exists.
// It panics on any other error.
func mustRegisterOrLoad(reg *featuregate.Registry, id string, stage featuregate.Stage, opts ...featuregate.RegisterOption) *featuregate.Gate {
	gate, err := reg.Register(id, stage, opts...)

	if errors.Is(err, featuregate.ErrAlreadyRegistered) {
		// Gate is already registered; find it.
		// Only a handful of feature gates are registered, so it's fine to iterate over all of them.
		reg.VisitAll(func(g *featuregate.Gate) {
			if g.ID() == id {
				gate = g
				return
			}
		})
	} else if err != nil {
		panic(err)
	}

	return gate
}
