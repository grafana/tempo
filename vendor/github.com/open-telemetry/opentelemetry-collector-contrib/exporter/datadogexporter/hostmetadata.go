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

package datadogexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata"
)

// newMetadataConfigfromConfig creates a new metadata pusher config from the main
func newMetadataConfigfromConfig(cfg *Config) hostmetadata.PusherConfig {
	return hostmetadata.PusherConfig{
		ConfigHostname:      cfg.Hostname,
		ConfigTags:          cfg.HostMetadata.Tags,
		MetricsEndpoint:     cfg.Metrics.Endpoint,
		APIKey:              string(cfg.API.Key),
		UseResourceMetadata: cfg.HostMetadata.HostnameSource == HostnameSourceFirstResource,
		InsecureSkipVerify:  cfg.TLSSetting.InsecureSkipVerify,
		TimeoutSettings:     cfg.TimeoutSettings,
		RetrySettings:       cfg.RetrySettings,
	}
}
