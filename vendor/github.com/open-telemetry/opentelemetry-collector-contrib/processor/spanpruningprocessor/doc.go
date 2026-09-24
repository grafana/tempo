// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate make mdatagen

// Package spanpruningprocessor detects duplicate or similar leaf spans within a
// single trace and replaces each set with a single aggregated summary span.
// Leaf spans are spans that are never referenced as a parent by another span.
// When all children of a parent are aggregated, the parent can also be
// aggregated, preserving the trace structure while reducing volume.
package spanpruningprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanpruningprocessor"
