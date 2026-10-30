// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package x

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	ecsdetector "go.opentelemetry.io/contrib/detectors/aws/ecs"
	eksdetector "go.opentelemetry.io/contrib/detectors/aws/eks"
	azurevmdetector "go.opentelemetry.io/contrib/detectors/azure/azurevm"
	gcpdetector "go.opentelemetry.io/contrib/detectors/gcp"

	"go.opentelemetry.io/contrib/otelconf/internal/kv"
)

func resourceOpts(detectors []ExperimentalResourceDetector) []resource.Option {
	opts := []resource.Option{}
	for _, d := range detectors {
		if d.AWSECS != nil {
			opts = append(opts, resource.WithDetectors(ecsdetector.NewResourceDetector()))
		}
		if d.AWSEKS != nil {
			opts = append(opts, resource.WithDetectors(eksdetector.NewResourceDetector()))
		}
		if d.AzureVM != nil {
			opts = append(opts, resource.WithDetectors(azurevmdetector.New()))
		}
		if d.GCP != nil {
			opts = append(opts, resource.WithDetectors(gcpdetector.NewDetector()))
		}
		if d.Container != nil {
			opts = append(opts, resource.WithContainer())
		}
		if d.Host != nil {
			opts = append(opts, resource.WithHost(), resource.WithOS())
		}
		if d.Process != nil {
			opts = append(opts, resource.WithProcess())
		}
		if d.Service != nil {
			opts = append(opts, resource.WithService())
		}
	}
	return opts
}

type resourceBuilder func(context.Context, ...resource.Option) (*resource.Resource, error)

func newResource(ctx context.Context, r *Resource) (*resource.Resource, error) {
	return newResourceWithBuilder(ctx, r, resource.New)
}

func newResourceWithBuilder(ctx context.Context, r *Resource, build resourceBuilder) (*resource.Resource, error) {
	if r == nil {
		return resource.DefaultWithContext(ctx), nil
	}

	attrs := make([]attribute.KeyValue, 0, len(r.Attributes))
	for _, v := range r.Attributes {
		attrs = append(attrs, kv.FromNameValue(v.Name, v.Value))
	}

	var schema string
	if r.SchemaUrl != nil {
		schema = *r.SchemaUrl
	}
	opts := []resource.Option{
		resource.WithAttributes(resource.DefaultWithContext(ctx).Attributes()...),
		resource.WithAttributes(attrs...),
		resource.WithSchemaURL(schema),
	}

	base, err := build(ctx, opts...)
	if err != nil {
		return nil, err
	}

	if r.DetectionDevelopment == nil {
		return base, nil
	}

	detected, err := newDetectedResource(ctx, r.DetectionDevelopment, build)
	if detected == nil {
		return base, err
	}

	merged, mergeErr := resource.Merge(detected, base)
	return merged, errors.Join(err, mergeErr)
}

func newDetectedResource(ctx context.Context, detection *ExperimentalResourceDetection, build resourceBuilder) (*resource.Resource, error) {
	filter, err := newIncludeExcludeFilter(detection.Attributes)
	if err != nil {
		return nil, err
	}

	opts := resourceOpts(detection.Detectors)
	if len(opts) == 0 {
		return resource.NewSchemaless(), nil
	}

	detected, err := build(ctx, opts...)
	if detected == nil {
		return nil, err
	}

	attrs := detected.Attributes()
	filtered := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		if filter(attr) {
			filtered = append(filtered, attr)
		}
	}
	if len(filtered) == 0 {
		return resource.NewSchemaless(), err
	}

	return resource.NewWithAttributes(detected.SchemaURL(), filtered...), err
}
