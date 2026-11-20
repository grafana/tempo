// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package encoding // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding"

import (
	"io"

	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	defaultFlushBytes = 1024 * 1024 // 1MB
	defaultFlushItems = 1000        // 1000 items
)

// LogsMarshalerExtension is an extension that marshals logs.
type LogsMarshalerExtension interface {
	extension.Extension
	plog.Marshaler
}

// LogsUnmarshalerExtension is an extension that unmarshals logs.
type LogsUnmarshalerExtension interface {
	extension.Extension
	plog.Unmarshaler
}

// LogsDecoder unmarshals logs from a stream, returning one batch per DecodeLogs call.
type LogsDecoder interface {
	// DecodeLogs is expected to be called iteratively to read all derived plog.Logs batches from the stream.
	// The last batch of logs should be returned with a nil error. io.EOF error should follow on the subsequent call.
	DecodeLogs() (plog.Logs, error)
	// Offset returns the offset after the most recent batch read from the stream, or the initial offset.
	// The exact meaning of the offset may vary by decoder (e.g. bytes, lines, records).
	// You may use this value with WithOffset option to resume reading from the same offset when retrying after a failure.
	Offset() int64
}

// LogsDecoderFactory creates LogsDecoder instances for streaming log deserialization.
type LogsDecoderFactory interface {
	NewLogsDecoder(reader io.Reader, options ...DecoderOption) (LogsDecoder, error)
}

// LogsDecoderExtension is an extension that unmarshals logs from a stream.
type LogsDecoderExtension interface {
	extension.Extension
	LogsDecoderFactory
}

// MetricsMarshalerExtension is an extension that marshals metrics.
type MetricsMarshalerExtension interface {
	extension.Extension
	pmetric.Marshaler
}

// MetricsUnmarshalerExtension is an extension that unmarshals metrics.
type MetricsUnmarshalerExtension interface {
	extension.Extension
	pmetric.Unmarshaler
}

// MetricsDecoder unmarshals metrics from a stream, returning one batch per DecodeMetrics call.
type MetricsDecoder interface {
	// DecodeMetrics is expected to be called iteratively to read all derived pmetric.Metrics batches from the stream.
	// The last batch of metrics should be returned with a nil error. io.EOF error should follow on the subsequent call.
	DecodeMetrics() (pmetric.Metrics, error)
	// Offset returns the offset after the most recent batch read from the stream, or the initial offset.
	// The exact meaning of the offset may vary by decoder (e.g. bytes, lines, records).
	// You may use this value with WithOffset option to resume reading from the same offset when retrying after a failure.
	Offset() int64
}

// MetricsDecoderFactory creates MetricsDecoder instances for streaming metric deserialization.
type MetricsDecoderFactory interface {
	NewMetricsDecoder(reader io.Reader, options ...DecoderOption) (MetricsDecoder, error)
}

// MetricsDecoderExtension is an extension that unmarshals metrics from a stream.
type MetricsDecoderExtension interface {
	extension.Extension
	MetricsDecoderFactory
}

// TracesMarshalerExtension is an extension that marshals traces.
type TracesMarshalerExtension interface {
	extension.Extension
	ptrace.Marshaler
}

// TracesUnmarshalerExtension is an extension that unmarshals traces.
type TracesUnmarshalerExtension interface {
	extension.Extension
	ptrace.Unmarshaler
}

// ProfilesMarshalerExtension is an extension that marshals profiles.
type ProfilesMarshalerExtension interface {
	extension.Extension
	pprofile.Marshaler
}

// ProfilesUnmarshalerExtension is an extension that unmarshals Profiles.
type ProfilesUnmarshalerExtension interface {
	extension.Extension
	pprofile.Unmarshaler
}

// DecoderOptions configures the behavior of stream decoding.
// FlushBytes and FlushItems control how often the decoder should flush decoded data from the stream.
// Offset defines the initial stream offset for the stream.
// Use NewDecoderOptions to construct with default options.
type DecoderOptions struct {
	FlushBytes int64
	FlushItems int64
	Offset     int64
}

// NewDecoderOptions applies [DecoderOption] values to the defaults for streaming
// decoders created by [LogsDecoderFactory] and [MetricsDecoderFactory].
// For message decoder extensions, use [NewDecodeConfig].
func NewDecoderOptions(opts ...DecoderOption) DecoderOptions {
	options := DecoderOptions{
		FlushBytes: defaultFlushBytes,
		FlushItems: defaultFlushItems,
		Offset:     0,
	}

	for _, o := range opts {
		o(&options)
	}
	return options
}

// DecoderOption configures streaming decoders created by [LogsDecoderFactory]
// and [MetricsDecoderFactory]. Streaming factories apply these options with [NewDecoderOptions].
// For message decoder extension interfaces, use [MessageDecodeOption].
type DecoderOption func(*DecoderOptions)

// WithFlushBytes sets the number of bytes after stream decoder should flush.
// Use WithFlushBytes(0) to disable flushing by byte count.
func WithFlushBytes(b int64) DecoderOption {
	return func(o *DecoderOptions) {
		o.FlushBytes = b
	}
}

// WithFlushItems sets the number of items after stream decoder should flush.
// Use WithFlushItems(0) to disable flushing by item count.
func WithFlushItems(i int64) DecoderOption {
	return func(o *DecoderOptions) {
		o.FlushItems = i
	}
}

// WithOffset defines the initial stream offset for the stream.
// The exact meaning of the offset may vary by decoder (e.g. bytes, lines, records).
func WithOffset(offset int64) DecoderOption {
	return func(o *DecoderOptions) {
		o.Offset = offset
	}
}
