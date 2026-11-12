// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/exp/maps"

	"go.opentelemetry.io/contrib/propagators/autoprop"
)

func newPropagator(p *Propagator) (propagation.TextMapPropagator, error) {
	if p == nil {
		return propagation.NewCompositeTextMapPropagator(), nil
	}

	n := len(p.Composite)
	if n == 0 && p.CompositeList == nil {
		return propagation.NewCompositeTextMapPropagator(), nil
	}

	names := map[string]struct{}{}
	for _, propagator := range p.Composite {
		if propagator.B3 != nil {
			names["b3"] = struct{}{}
		}
		if propagator.B3Multi != nil {
			names["b3multi"] = struct{}{}
		}
		if propagator.Baggage != nil {
			names["baggage"] = struct{}{}
		}
		if propagator.Jaeger != nil {
			names["jaeger"] = struct{}{}
		}
		if propagator.Ottrace != nil {
			names["ottrace"] = struct{}{}
		}
		if propagator.Tracecontext != nil {
			names["tracecontext"] = struct{}{}
		}
	}

	if p.CompositeList != nil {
		for v := range strings.SplitSeq(*p.CompositeList, ",") {
			names[v] = struct{}{}
		}
	}

	if len(names) == 0 {
		return autoprop.NewTextMapPropagator(), nil
	}

	return autoprop.TextMapPropagator(maps.Keys(names)...)
}
