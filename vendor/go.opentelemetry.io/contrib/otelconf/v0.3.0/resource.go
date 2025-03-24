// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

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

func newResource(res *Resource) *resource.Resource {
	if res == nil {
		return resource.Default()
	}

	var attrs []attribute.KeyValue
	for _, v := range res.Attributes {
		attrs = append(attrs, keyVal(v.Name, v.Value))
	}

	if res.SchemaUrl == nil {
		return resource.NewSchemaless(attrs...)
	}
	return resource.NewWithAttributes(*res.SchemaUrl, attrs...)
}
