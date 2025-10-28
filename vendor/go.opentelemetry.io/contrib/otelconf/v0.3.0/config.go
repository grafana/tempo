// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otelconf provides an OpenTelemetry declarative configuration SDK.
package otelconf // import "go.opentelemetry.io/contrib/otelconf/v0.3.0"

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	yaml "go.yaml.in/yaml/v3"
)

const (
	protocolProtobufHTTP = "http/protobuf"
	protocolProtobufGRPC = "grpc"

	compressionGzip = "gzip"
	compressionNone = "none"
)

type configOptions struct {
	ctx                   context.Context
	opentelemetryConfig   OpenTelemetryConfiguration
	loggerProviderOptions []sdklog.LoggerProviderOption
	meterProviderOptions  []sdkmetric.Option
	tracerProviderOptions []sdktrace.TracerProviderOption
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
	loggerProvider log.LoggerProvider
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

// LoggerProvider returns a configured log.LoggerProvider.
func (s *SDK) LoggerProvider() log.LoggerProvider {
	return s.loggerProvider
}

// Shutdown calls shutdown on all configured providers.
func (s *SDK) Shutdown(ctx context.Context) error {
	return s.shutdown(ctx)
}

var noopSDK = SDK{
	loggerProvider: nooplog.LoggerProvider{},
	meterProvider:  noopmetric.MeterProvider{},
	tracerProvider: nooptrace.TracerProvider{},
	shutdown:       func(context.Context) error { return nil },
}

// NewSDK creates SDK providers based on the configuration model.
func NewSDK(opts ...ConfigurationOption) (SDK, error) {
	o := configOptions{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		o = opt.apply(o)
	}
	if o.opentelemetryConfig.Disabled != nil && *o.opentelemetryConfig.Disabled {
		return noopSDK, nil
	}

	r := newResource(o.opentelemetryConfig.Resource)

	mp, mpShutdown, err := meterProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	tp, tpShutdown, err := tracerProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	lp, lpShutdown, err := loggerProvider(o, r)
	if err != nil {
		return noopSDK, err
	}

	return SDK{
		meterProvider:  mp,
		tracerProvider: tp,
		loggerProvider: lp,
		shutdown: func(ctx context.Context) error {
			return errors.Join(mpShutdown(ctx), tpShutdown(ctx), lpShutdown(ctx))
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

// WithLoggerProviderOptions appends LoggerProviderOptions used for constructing
// the LoggerProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithLoggerProviderOptions(opts ...sdklog.LoggerProviderOption) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.loggerProviderOptions = append(c.loggerProviderOptions, opts...)
		return c
	})
}

// WithMeterProviderOptions appends metric.Options used for constructing the
// MeterProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithMeterProviderOptions(opts ...sdkmetric.Option) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.meterProviderOptions = append(c.meterProviderOptions, opts...)
		return c
	})
}

// WithTracerProviderOptions appends TracerProviderOptions used for constructing
// the TracerProvider. OpenTelemetryConfiguration takes precedence over these options.
func WithTracerProviderOptions(opts ...sdktrace.TracerProviderOption) ConfigurationOption {
	return configurationOptionFunc(func(c configOptions) configOptions {
		c.tracerProviderOptions = append(c.tracerProviderOptions, opts...)
		return c
	})
}

// ParseYAML parses a YAML configuration file into an OpenTelemetryConfiguration.
func ParseYAML(file []byte) (*OpenTelemetryConfiguration, error) {
	var cfg OpenTelemetryConfiguration
	err := yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// createTLSConfig creates a tls.Config from certificate files.
func createTLSConfig(caCertFile, clientCertFile, clientKeyFile *string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if caCertFile != nil {
		caText, err := os.ReadFile(*caCertFile)
		if err != nil {
			return nil, err
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caText) {
			return nil, errors.New("could not create certificate authority chain from certificate")
		}
		tlsConfig.RootCAs = certPool
	}
	if clientCertFile != nil {
		if clientKeyFile == nil {
			return nil, errors.New("client certificate was provided but no client key was provided")
		}
		clientCert, err := tls.LoadX509KeyPair(*clientCertFile, *clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not use client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	return tlsConfig, nil
}

// createHeadersConfig combines the two header config fields. Headers take precedence over headersList.
func createHeadersConfig(headers []NameStringValuePair, headersList *string) (map[string]string, error) {
	result := make(map[string]string)
	if headersList != nil {
		// Parsing follows https://github.com/open-telemetry/opentelemetry-configuration/blob/568e5080816d40d75792eb754fc96bde09654159/schema/type_descriptions.yaml#L584.
		headerslist, err := baggage.Parse(*headersList)
		if err != nil {
			return nil, fmt.Errorf("invalid headers list: %w", err)
		}
		for _, kv := range headerslist.Members() {
			result[kv.Key()] = kv.Value()
		}
	}
	// Headers take precedence over HeadersList, so this has to be after HeadersList is processed
	if len(headers) > 0 {
		for _, kv := range headers {
			if kv.Value != nil {
				result[kv.Name] = *kv.Value
			}
		}
	}
	return result, nil
}
