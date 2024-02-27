// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config // import "go.opentelemetry.io/contrib/config"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc/protobuf"

	compressionGzip = "gzip"
	compressionNone = "none"
)

type configOptions struct {
	ctx                 context.Context
	opentelemetryConfig OpenTelemetryConfiguration
}

type shutdownFunc func(context.Context) error

func noopShutdown(context.Context) error {
	return nil
}

// SDK is a struct that contains all the providers
// configured via the configuration model.
type SDK struct {
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	shutdown       shutdownFunc
}

// TracerProvider returns a configured trace.TracerProvider.
func (s *SDK) TracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

// MeterProvider returns a configured metric.MeterProvider.
func (s *SDK) MeterProvider() metric.MeterProvider {
	return s.meterProvider
}

// Shutdown calls shutdown on all configured providers.
func (s *SDK) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

// NewSDK creates SDK providers based on the configuration model.
//
// Caution: The implementation only returns noop providers.
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	o := configOptions{}
	for _, opt := range opts {
		o = opt.apply(o)
	}

	r, err := newResource(o.opentelemetryConfig.Resource)
	if err != nil {
		return SDK{}, err
	}

	mp, mpShutdown := initMeterProvider(o)
	tp, tpShutdown, err := tracerProvider(o, r)
	if err != nil {
		return SDK{}, err
	}

	return SDK{
		meterProvider:  mp,
		tracerProvider: tp,
		shutdown: func(ctx context.Context) error {
			err := mpShutdown(ctx)
			return errors.Join(err, tpShutdown(ctx))
		},
	}, nil
}

// ConfigurationOption configures options for providers.
type ConfigurationOption interface {
	apply(configOptions) configOptions
}

type configurationOptionFunc func(configOptions) configOptions

func (fn configurationOptionFunc) apply(cfg configOptions) configOptions {
	return fn(cfg)
}

// WithContext sets the context.Context for the SDK.
func WithContext(ctx context.Context) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.ctx = ctx
		return c
	})
}

// WithOpenTelemetryConfiguration sets the OpenTelemetryConfiguration used
// to produce the SDK.
func WithOpenTelemetryConfiguration(cfg OpenTelemetryConfiguration) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.opentelemetryConfig = cfg
		return c
	})
}

// TODO: implement parsing functionality:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4373
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4412

// TODO: create SDK from the model:
// - https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4371
