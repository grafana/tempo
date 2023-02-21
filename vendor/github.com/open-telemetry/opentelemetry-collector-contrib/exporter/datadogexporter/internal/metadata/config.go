// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/metadata"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// PusherConfig is the configuration for the metadata pusher goroutine.
type PusherConfig struct {
	// ConfigHosthame is the hostname set in the configuration of the exporter (empty if unset).
	ConfigHostname string
	// ConfigTags are the tags set in the configuration of the exporter (empty if unset).
	ConfigTags []string
	// MetricsEndpoint is the metrics endpoint.
	MetricsEndpoint string
	// APIKey is the API key set in configuration.
	APIKey string
	// UseResourceMetadata is the value of 'use_resource_metadata' on the top-level configuration.
	UseResourceMetadata bool
	// InsecureSkipVerify is the value of `tls.insecure_skip_verify` on the configuration.
	InsecureSkipVerify bool
	// TimeoutSettings of exporter.
	TimeoutSettings exporterhelper.TimeoutSettings
	// RetrySettings of exporter.
	RetrySettings exporterhelper.RetrySettings
}
