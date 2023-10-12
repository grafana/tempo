// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensusreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
)

// Config defines configuration for OpenCensus receiver.
type Config struct {
	// Configures the receiver server protocol.
	configgrpc.GRPCServerSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct

	// CorsOrigins are the allowed CORS origins for HTTP/JSON requests to grpc-gateway adapter
	// for the OpenCensus receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`
}

func (cfg *Config) buildOptions() []ocOption {
	var opts []ocOption
	if len(cfg.CorsOrigins) > 0 {
		opts = append(opts, withCorsOrigins(cfg.CorsOrigins))
	}

	opts = append(opts, withGRPCServerSettings(cfg.GRPCServerSettings))

	return opts
}

var _ component.Config = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	return nil
}
