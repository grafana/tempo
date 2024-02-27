// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func newResource(res *Resource) (*resource.Resource, error) {
	if res == nil || res.Attributes == nil {
		return resource.Default(), nil
	}
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(*res.SchemaUrl,
			semconv.ServiceName(*res.Attributes.ServiceName),
		))
}
