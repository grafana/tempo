// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration settings for the Zipkin exporter.
type Config struct {
	QueueSettings             exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
	configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// Configures the exporter client.
	// The Endpoint to send the Zipkin trace data to (e.g.: http://some.url:9411/api/v2/spans).
	confighttp.ClientConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	Format string `mapstructure:"format"`

	DefaultServiceName string `mapstructure:"default_service_name"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errors.New("endpoint required")
	}
	return nil
}
