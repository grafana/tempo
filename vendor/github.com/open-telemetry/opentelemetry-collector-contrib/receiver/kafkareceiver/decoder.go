// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"context"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding"
)

type messageDecoder[T any] func(context.Context, []byte) (T, error)

// newLogsDecoder prefers the explicit message API when an extension implements it.
// Existing encodings retain their byte-only unmarshaling behavior.
func newLogsDecoder(name string, set receiver.Settings, host component.Host) (messageDecoder[plog.Logs], error) {
	if decoder, err := loadEncodingExtension[encoding.LogsMessageDecoderExtension](host, name, "logs"); err == nil {
		return func(ctx context.Context, data []byte) (plog.Logs, error) {
			return decoder.DecodeLogs(ctx, data, encoding.WithMessageMetadata(client.FromContext(ctx).Metadata))
		}, nil
	}
	unmarshaler, err := newLogsUnmarshaler(name, set, host)
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, data []byte) (plog.Logs, error) {
		return unmarshaler.UnmarshalLogs(data)
	}, nil
}

// newMetricsDecoder prefers the explicit message API when an extension implements it.
// Existing encodings retain their byte-only unmarshaling behavior.
func newMetricsDecoder(name string, set receiver.Settings, host component.Host) (messageDecoder[pmetric.Metrics], error) {
	if decoder, err := loadEncodingExtension[encoding.MetricsMessageDecoderExtension](host, name, "metrics"); err == nil {
		return func(ctx context.Context, data []byte) (pmetric.Metrics, error) {
			return decoder.DecodeMetrics(ctx, data, encoding.WithMessageMetadata(client.FromContext(ctx).Metadata))
		}, nil
	}
	unmarshaler, err := newMetricsUnmarshaler(name, set, host)
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, data []byte) (pmetric.Metrics, error) {
		return unmarshaler.UnmarshalMetrics(data)
	}, nil
}

// newTracesDecoder prefers the explicit message API when an extension implements it.
// Existing encodings retain their byte-only unmarshaling behavior.
func newTracesDecoder(name string, set receiver.Settings, host component.Host) (messageDecoder[ptrace.Traces], error) {
	if decoder, err := loadEncodingExtension[encoding.TracesMessageDecoderExtension](host, name, "traces"); err == nil {
		return func(ctx context.Context, data []byte) (ptrace.Traces, error) {
			return decoder.DecodeTraces(ctx, data, encoding.WithMessageMetadata(client.FromContext(ctx).Metadata))
		}, nil
	}
	unmarshaler, err := newTracesUnmarshaler(name, set, host)
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, data []byte) (ptrace.Traces, error) {
		return unmarshaler.UnmarshalTraces(data)
	}, nil
}

// newProfilesDecoder prefers the explicit message API when an extension implements it.
// Existing encodings retain their byte-only unmarshaling behavior.
func newProfilesDecoder(name string, set receiver.Settings, host component.Host) (messageDecoder[pprofile.Profiles], error) {
	if decoder, err := loadEncodingExtension[encoding.ProfilesMessageDecoderExtension](host, name, "profiles"); err == nil {
		return func(ctx context.Context, data []byte) (pprofile.Profiles, error) {
			return decoder.DecodeProfiles(ctx, data, encoding.WithMessageMetadata(client.FromContext(ctx).Metadata))
		}, nil
	}
	unmarshaler, err := newProfilesUnmarshaler(name, set, host)
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, data []byte) (pprofile.Profiles, error) {
		return unmarshaler.UnmarshalProfiles(data)
	}, nil
}
