// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterset // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"

// FilterSet is an interface for matching strings against a set of filters.
type FilterSet interface {
	// Matches returns true if the given string matches at least one
	// of the filters encapsulated by the FilterSet.
	Matches(string) bool
}
