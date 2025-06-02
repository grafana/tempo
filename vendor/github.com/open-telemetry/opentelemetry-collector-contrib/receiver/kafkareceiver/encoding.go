// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver/internal/unmarshaler"
)

var (
	errUnknownEncodingExtension = errors.New("unknown encoding extension")
	errInvalidComponentType     = errors.New("invalid component type")
)

func newTracesUnmarshaler(encoding string, _ receiver.Settings, host component.Host) (ptrace.Unmarshaler, error) {
	// Extensions take precedence.
	if unmarshaler, err := loadEncodingExtension[ptrace.Unmarshaler](host, encoding, "traces"); err != nil {
		if !errors.Is(err, errInvalidComponentType) && !errors.Is(err, errUnknownEncodingExtension) {
			return nil, err
		}
	} else {
		return unmarshaler, nil
	}
	switch encoding {
	case "otlp_proto":
		return &ptrace.ProtoUnmarshaler{}, nil
	case "otlp_json":
		return &ptrace.JSONUnmarshaler{}, nil
	case "jaeger_proto":
		return unmarshaler.JaegerProtoSpanUnmarshaler{}, nil
	case "jaeger_json":
		return unmarshaler.JaegerJSONSpanUnmarshaler{}, nil
	case "zipkin_proto":
		return zipkinv2.NewProtobufTracesUnmarshaler(false, false), nil
	case "zipkin_json":
		return zipkinv2.NewJSONTracesUnmarshaler(false), nil
	case "zipkin_thrift":
		return zipkinv1.NewThriftTracesUnmarshaler(), nil
	}
	return nil, fmt.Errorf("unrecognized traces encoding %q", encoding)
}

func newLogsUnmarshaler(encoding string, set receiver.Settings, host component.Host) (plog.Unmarshaler, error) {
	// Extensions take precedence.
	if unmarshaler, err := loadEncodingExtension[plog.Unmarshaler](host, encoding, "logs"); err != nil {
		if !errors.Is(err, errInvalidComponentType) && !errors.Is(err, errUnknownEncodingExtension) {
			return nil, err
		}
	} else {
		return unmarshaler, nil
	}
	switch encoding {
	case "otlp_proto":
		return &plog.ProtoUnmarshaler{}, nil
	case "otlp_json":
		return &plog.JSONUnmarshaler{}, nil
	case "raw":
		return unmarshaler.RawLogsUnmarshaler{}, nil
	case "json":
		return unmarshaler.JSONLogsUnmarshaler{}, nil
	case "azure_resource_logs":
		return &azure.ResourceLogsUnmarshaler{
			Version: set.BuildInfo.Version,
			Logger:  set.Logger,
		}, nil
	case "text":
		return unmarshaler.NewTextLogsUnmarshaler("utf-8")
	}
	// There is a special case for text-based encodings, where you can specify
	// the text encoding (e.g. utf8, utf16) as a suffix in the encoding name.
	if textEncodingName, ok := strings.CutPrefix(encoding, "text_"); ok {
		u, err := unmarshaler.NewTextLogsUnmarshaler(textEncodingName)
		if err != nil {
			return nil, fmt.Errorf("invalid text encoding: %w", err)
		}
		return u, nil
	}
	return nil, fmt.Errorf("unrecognized logs encoding %q", encoding)
}

func newMetricsUnmarshaler(encoding string, _ receiver.Settings, host component.Host) (pmetric.Unmarshaler, error) {
	// Extensions take precedence.
	if unmarshaler, err := loadEncodingExtension[pmetric.Unmarshaler](host, encoding, "metrics"); err != nil {
		if !errors.Is(err, errInvalidComponentType) && !errors.Is(err, errUnknownEncodingExtension) {
			return nil, err
		}
	} else {
		return unmarshaler, nil
	}
	switch encoding {
	case "otlp_proto":
		return &pmetric.ProtoUnmarshaler{}, nil
	case "otlp_json":
		return &pmetric.JSONUnmarshaler{}, nil
	}
	return nil, fmt.Errorf("unrecognized metrics encoding %q", encoding)
}

// loadEncodingExtension tries to load an available extension for the given encoding.
func loadEncodingExtension[T any](host component.Host, encoding, signalType string) (T, error) {
	var zero T
	extensionID, err := encodingToComponentID(encoding)
	if err != nil {
		return zero, err
	}
	encodingExtension, ok := host.GetExtensions()[*extensionID]
	if !ok {
		return zero, fmt.Errorf("invalid encoding %q: %w", encoding, errUnknownEncodingExtension)
	}
	unmarshaler, ok := encodingExtension.(T)
	if !ok {
		return zero, fmt.Errorf("extension %q is not a %s unmarshaler", encoding, signalType)
	}
	return unmarshaler, nil
}

// encodingToComponentID attempts to parse the encoding string as a component ID.
func encodingToComponentID(encoding string) (*component.ID, error) {
	var id component.ID
	if err := id.UnmarshalText([]byte(encoding)); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidComponentType, err)
	}
	return &id, nil
}
