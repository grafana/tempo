// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafka // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka"

// SignalHeaderKey is the Kafka record header that identifies the telemetry
// signal. The name matches the collector's otelcol.signal attribute. Values
// are pipeline.Signal.String(): logs, metrics, traces, or profiles.
const SignalHeaderKey = "otelcol.signal"
