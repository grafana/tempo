// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config/v0.2.0"

import (
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func keyVal(k string, v any) attribute.KeyValue {
	switch val := v.(type) {
	case bool:
		return attribute.Bool(k, val)
	case int64:
		return attribute.Int64(k, val)
	case uint64:
		return attribute.String(k, strconv.FormatUint(val, 10))
	case float64:
		return attribute.Float64(k, val)
	case int8:
		return attribute.Int64(k, int64(val))
	case uint8:
		return attribute.Int64(k, int64(val))
	case int16:
		return attribute.Int64(k, int64(val))
	case uint16:
		return attribute.Int64(k, int64(val))
	case int32:
		return attribute.Int64(k, int64(val))
	case uint32:
		return attribute.Int64(k, int64(val))
	case float32:
		return attribute.Float64(k, float64(val))
	case int:
		return attribute.Int(k, val)
	case uint:
		return attribute.String(k, strconv.FormatUint(uint64(val), 10))
	case string:
		return attribute.String(k, val)
	default:
		return attribute.String(k, fmt.Sprint(v))
	}
}

func newResource(res *Resource) (*resource.Resource, error) {
	if res == nil || res.Attributes == nil {
		return resource.Default(), nil
	}
	var attrs []attribute.KeyValue

	for k, v := range res.Attributes {
		attrs = append(attrs, keyVal(k, v))
	}

	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(*res.SchemaUrl,
			attrs...,
		))
}
