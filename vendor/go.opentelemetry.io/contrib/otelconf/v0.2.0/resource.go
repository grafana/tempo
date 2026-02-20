// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.2.0"

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func newResource(res *Resource) (*resource.Resource, error) {
	if res == nil || res.Attributes == nil {
		return resource.Default(), nil
	}
	attrs := make([]attribute.KeyValue, 0, len(res.Attributes))

	for k, v := range res.Attributes {
		attrs = append(attrs, kv.FromNameValue(k, v))
	}

	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(*res.SchemaUrl,
			attrs...,
		))
}
