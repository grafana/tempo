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
	"encoding"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/hostmetadata/valid"
)

var (
	errUnsetAPIKey   = errors.New("api.key is not set")
	errNoMetadata    = errors.New("only_metadata can't be enabled when host_metadata::enabled = false or host_metadata::hostname_source != first_resource")
	errEmptyEndpoint = errors.New("endpoint cannot be empty")
)

const (
	// DefaultSite is the default site of the Datadog intake to send data to
	DefaultSite = "datadoghq.com"
)

// APIConfig defines the API configuration options
type APIConfig struct {
	// Key is the Datadog API key to associate your Agent's data with your organization.
	// Create a new API key here: https://app.datadoghq.com/account/settings
	Key configopaque.String `mapstructure:"key"`

	// Site is the site of the Datadog intake to send data to.
	// The default value is "datadoghq.com".
	Site string `mapstructure:"site"`

	// FailOnInvalidKey states whether to exit at startup on invalid API key.
	// The default value is false.
	FailOnInvalidKey bool `mapstructure:"fail_on_invalid_key"`
}

// MetricsConfig defines the metrics exporter specific configuration options
type MetricsConfig struct {
	// DeltaTTL defines the time that previous points of a cumulative monotonic
	// metric are kept in memory to calculate deltas
	DeltaTTL int64 `mapstructure:"delta_ttl"`

	// TCPAddr.Endpoint is the host of the Datadog intake server to send metrics to.
	// If unset, the value is obtained from the Site.
	confignet.TCPAddr `mapstructure:",squash"`

	ExporterConfig MetricsExporterConfig `mapstructure:",squash"`

	// HistConfig defines the export of OTLP Histograms.
	HistConfig HistogramConfig `mapstructure:"histograms"`

	// SumConfig defines the export of OTLP Sums.
	SumConfig SumConfig `mapstructure:"sums"`

	// SummaryConfig defines the export for OTLP Summaries.
	SummaryConfig SummaryConfig `mapstructure:"summaries"`
}

type HistogramMode string

const (
	// HistogramModeNoBuckets reports no bucket histogram metrics. .sum and .count metrics will still be sent
	// if `send_count_sum_metrics` is enabled.
	HistogramModeNoBuckets HistogramMode = "nobuckets"
	// HistogramModeCounters reports histograms as Datadog counts, one metric per bucket.
	HistogramModeCounters HistogramMode = "counters"
	// HistogramModeDistributions reports histograms as Datadog distributions (recommended).
	HistogramModeDistributions HistogramMode = "distributions"
)

var _ encoding.TextUnmarshaler = (*HistogramMode)(nil)

func (hm *HistogramMode) UnmarshalText(in []byte) error {
	switch mode := HistogramMode(in); mode {
	case HistogramModeCounters, HistogramModeDistributions, HistogramModeNoBuckets:
		*hm = mode
		return nil
	default:
		return fmt.Errorf("invalid histogram mode %q", mode)
	}
}

// HistogramConfig customizes export of OTLP Histograms.
type HistogramConfig struct {
	// Mode for exporting histograms. Valid values are 'distributions', 'counters' or 'nobuckets'.
	//  - 'distributions' sends histograms as Datadog distributions (recommended).
	//  - 'counters' sends histograms as Datadog counts, one metric per bucket.
	//  - 'nobuckets' sends no bucket histogram metrics. .sum and .count metrics will still be sent
	//    if `send_count_sum_metrics` is enabled.
	//
	// The current default is 'distributions'.
	Mode HistogramMode `mapstructure:"mode"`

	// SendCountSum states if the export should send .sum and .count metrics for histograms.
	// The current default is false.
	SendCountSum bool `mapstructure:"send_count_sum_metrics"`
}

func (c *HistogramConfig) validate() error {
	if c.Mode == HistogramModeNoBuckets && !c.SendCountSum {
		return fmt.Errorf("'nobuckets' mode and `send_count_sum_metrics` set to false will send no histogram metrics")
	}
	return nil
}

// CumulativeMonotonicSumMode is the export mode for OTLP Sum metrics.
type CumulativeMonotonicSumMode string

const (
	// CumulativeMonotonicSumModeToDelta calculates delta for
	// cumulative monotonic sum metrics in the client side and reports
	// them as Datadog counts.
	CumulativeMonotonicSumModeToDelta CumulativeMonotonicSumMode = "to_delta"

	// CumulativeMonotonicSumModeRawValue reports the raw value for
	// cumulative monotonic sum metrics as a Datadog gauge.
	CumulativeMonotonicSumModeRawValue CumulativeMonotonicSumMode = "raw_value"
)

var _ encoding.TextUnmarshaler = (*CumulativeMonotonicSumMode)(nil)

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (sm *CumulativeMonotonicSumMode) UnmarshalText(in []byte) error {
	switch mode := CumulativeMonotonicSumMode(in); mode {
	case CumulativeMonotonicSumModeToDelta,
		CumulativeMonotonicSumModeRawValue:
		*sm = mode
		return nil
	default:
		return fmt.Errorf("invalid cumulative monotonic sum mode %q", mode)
	}
}

// SumConfig customizes export of OTLP Sums.
type SumConfig struct {
	// CumulativeMonotonicMode is the mode for exporting OTLP Cumulative Monotonic Sums.
	// Valid values are 'to_delta' or 'raw_value'.
	//  - 'to_delta' calculates delta for cumulative monotonic sums and sends it as a Datadog count.
	//  - 'raw_value' sends the raw value of cumulative monotonic sums as Datadog gauges.
	//
	// The default is 'to_delta'.
	// See https://docs.datadoghq.com/metrics/otlp/?tab=sum#mapping for details and examples.
	CumulativeMonotonicMode CumulativeMonotonicSumMode `mapstructure:"cumulative_monotonic_mode"`
}

// SummaryMode is the export mode for OTLP Summary metrics.
type SummaryMode string

const (
	// SummaryModeNoQuantiles sends no `.quantile` metrics. `.sum` and `.count` metrics will still be sent.
	SummaryModeNoQuantiles SummaryMode = "noquantiles"
	// SummaryModeGauges sends `.quantile` metrics as gauges tagged by the quantile.
	SummaryModeGauges SummaryMode = "gauges"
)

var _ encoding.TextUnmarshaler = (*SummaryMode)(nil)

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (sm *SummaryMode) UnmarshalText(in []byte) error {
	switch mode := SummaryMode(in); mode {
	case SummaryModeNoQuantiles,
		SummaryModeGauges:
		*sm = mode
		return nil
	default:
		return fmt.Errorf("invalid summary mode %q", mode)
	}
}

// SummaryConfig customizes export of OTLP Summaries.
type SummaryConfig struct {
	// Mode is the the mode for exporting OTLP Summaries.
	// Valid values are 'noquantiles' or 'gauges'.
	//  - 'noquantiles' sends no `.quantile` metrics. `.sum` and `.count` metrics will still be sent.
	//  - 'gauges' sends `.quantile` metrics as gauges tagged by the quantile.
	//
	// The default is 'gauges'.
	// See https://docs.datadoghq.com/metrics/otlp/?tab=summary#mapping for details and examples.
	Mode SummaryMode `mapstructure:"mode"`
}

// MetricsExporterConfig provides options for a user to customize the behavior of the
// metrics exporter
type MetricsExporterConfig struct {
	// ResourceAttributesAsTags, if set to true, will use the exporterhelper feature to transform all
	// resource attributes into metric labels, which are then converted into tags
	ResourceAttributesAsTags bool `mapstructure:"resource_attributes_as_tags"`

	// InstrumentationScopeMetadataAsTags, if set to true, adds the name and version of the
	// instrumentation scope that created a metric to the metric tags
	InstrumentationScopeMetadataAsTags bool `mapstructure:"instrumentation_scope_metadata_as_tags"`
}

// TracesConfig defines the traces exporter specific configuration options
type TracesConfig struct {
	// TCPAddr.Endpoint is the host of the Datadog intake server to send traces to.
	// If unset, the value is obtained from the Site.
	confignet.TCPAddr `mapstructure:",squash"`

	// ignored resources
	// A blacklist of regular expressions can be provided to disable certain traces based on their resource name
	// all entries must be surrounded by double quotes and separated by commas.
	// ignore_resources: ["(GET|POST) /healthcheck"]
	IgnoreResources []string `mapstructure:"ignore_resources"`

	// SpanNameRemappings is the map of datadog span names and preferred name to map to. This can be used to
	// automatically map Datadog Span Operation Names to an updated value. All entries should be key/value pairs.
	// span_name_remappings:
	//   io.opentelemetry.javaagent.spring.client: spring.client
	//   instrumentation:express.server: express
	//   go.opentelemetry.io_contrib_instrumentation_net_http_otelhttp.client: http.client
	SpanNameRemappings map[string]string `mapstructure:"span_name_remappings"`

	// If set to true the OpenTelemetry span name will used in the Datadog resource name.
	// If set to false the resource name will be filled with the instrumentation library name + span kind.
	// The default value is `false`.
	SpanNameAsResourceName bool `mapstructure:"span_name_as_resource_name"`

	// flushInterval defines the interval in seconds at which the writer flushes traces
	// to the intake; used in tests.
	flushInterval float64
}

// LogsConfig defines logs exporter specific configuration
type LogsConfig struct {
	// TCPAddr.Endpoint is the host of the Datadog intake server to send logs to.
	// If unset, the value is obtained from the Site.
	confignet.TCPAddr `mapstructure:",squash"`

	// DumpPayloads report whether payloads should be dumped when logging level is debug.
	DumpPayloads bool `mapstructure:"dump_payloads"`
}

// TagsConfig defines the tag-related configuration
// It is embedded in the configuration
type TagsConfig struct {
	// Hostname is the host name for unified service tagging.
	// If unset, it is determined automatically.
	Hostname string `mapstructure:"hostname"`
}

// HostnameSource is the source for the hostname of host metadata.
type HostnameSource string

const (
	// HostnameSourceFirstResource picks the host metadata hostname from the resource
	// attributes on the first OTLP payload that gets to the exporter. If it is lacking any
	// hostname-like attributes, it will fallback to 'config_or_system' behavior (see below).
	//
	// Do not use this hostname source if receiving data from multiple hosts.
	HostnameSourceFirstResource HostnameSource = "first_resource"

	// HostnameSourceConfigOrSystem picks the host metadata hostname from the 'hostname' setting,
	// and if this is empty, from available system APIs and cloud provider endpoints.
	HostnameSourceConfigOrSystem HostnameSource = "config_or_system"
)

var _ encoding.TextUnmarshaler = (*HostnameSource)(nil)

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (sm *HostnameSource) UnmarshalText(in []byte) error {
	switch mode := HostnameSource(in); mode {
	case HostnameSourceFirstResource,
		HostnameSourceConfigOrSystem:
		*sm = mode
		return nil
	default:
		return fmt.Errorf("invalid host metadata hostname source %q", mode)
	}
}

// HostMetadataConfig defines the host metadata related configuration.
// Host metadata is the information used for populating the infrastructure list,
// the host map and providing host tags functionality.
//
// The exporter will send host metadata for a single host, whose name is chosen
// according to `host_metadata::hostname_source`.
type HostMetadataConfig struct {
	// Enabled enables the host metadata functionality.
	Enabled bool `mapstructure:"enabled"`

	// HostnameSource is the source for the hostname of host metadata.
	// Valid values are 'first_resource' and 'config_or_system':
	// - 'first_resource' picks the host metadata hostname from the resource
	//    attributes on the first OTLP payload that gets to the exporter.
	//    If the first payload lacks hostname-like attributes, it will fallback to 'config_or_system'.
	//    Do not use this hostname source if receiving data from multiple hosts.
	// - 'config_or_system' picks the host metadata hostname from the 'hostname' setting,
	//    If this is empty it will use available system APIs and cloud provider endpoints.
	//
	// The default is 'config_or_system'.
	HostnameSource HostnameSource `mapstructure:"hostname_source"`

	// Tags is a list of host tags.
	// These tags will be attached to telemetry signals that have the host metadata hostname.
	// To attach tags to telemetry signals regardless of the host, use a processor instead.
	Tags []string `mapstructure:"tags"`
}

// LimitedTLSClientSetting is a subset of TLSClientSetting, see LimitedHTTPClientSettings for more details
type LimitedTLSClientSettings struct {
	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name.
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

type LimitedHTTPClientSettings struct {
	TLSSetting LimitedTLSClientSettings `mapstructure:"tls,omitempty"`
}

// Config defines configuration for the Datadog exporter.
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`

	LimitedHTTPClientSettings `mapstructure:",squash"`

	TagsConfig `mapstructure:",squash"`

	// API defines the Datadog API configuration.
	API APIConfig `mapstructure:"api"`

	// Metrics defines the Metrics exporter specific configuration
	Metrics MetricsConfig `mapstructure:"metrics"`

	// Traces defines the Traces exporter specific configuration
	Traces TracesConfig `mapstructure:"traces"`

	// Logs defines the Logs exporter specific configuration
	Logs LogsConfig `mapstructure:"logs"`

	// HostMetadata defines the host metadata specific configuration
	HostMetadata HostMetadataConfig `mapstructure:"host_metadata"`

	// OnlyMetadata defines whether to only send metadata
	// This is useful for agent-collector setups, so that
	// metadata about a host is sent to the backend even
	// when telemetry data is reported via a different host.
	//
	// This flag is incompatible with disabling host metadata,
	// `use_resource_metadata`, or `host_metadata::hostname_source != first_resource`
	OnlyMetadata bool `mapstructure:"only_metadata"`
}

var _ component.Config = (*Config)(nil)

// Validate the configuration for errors. This is required by component.Config.
func (c *Config) Validate() error {
	if c.OnlyMetadata && (!c.HostMetadata.Enabled || c.HostMetadata.HostnameSource != HostnameSourceFirstResource) {
		return errNoMetadata
	}

	if err := valid.Hostname(c.Hostname); c.Hostname != "" && err != nil {
		return fmt.Errorf("hostname field is invalid: %w", err)
	}

	if c.API.Key == "" {
		return errUnsetAPIKey
	}

	if c.Traces.IgnoreResources != nil {
		for _, entry := range c.Traces.IgnoreResources {
			_, err := regexp.Compile(entry)
			if err != nil {
				return fmt.Errorf("'%s' is not valid resource filter regular expression", entry)
			}
		}
	}

	if c.Traces.SpanNameRemappings != nil {
		for key, value := range c.Traces.SpanNameRemappings {
			if value == "" {
				return fmt.Errorf("'%s' is not valid value for span name remapping", value)
			}
			if key == "" {
				return fmt.Errorf("'%s' is not valid key for span name remapping", key)
			}
		}
	}

	err := c.Metrics.HistConfig.validate()
	if err != nil {
		return err
	}

	return nil
}

var _ error = (*renameError)(nil)

// renameError is an error related to a renamed setting.
type renameError struct {
	// oldName of the configuration option.
	oldName string
	// newName of the configuration option.
	newName string
	// issueNumber on opentelemetry-collector-contrib for tracking
	issueNumber uint
}

// List of settings that have been removed, but for which we keep a custom error.
var removedSettings = []renameError{
	{
		oldName:     "metrics::send_monotonic_counter",
		newName:     "metrics::sums::cumulative_monotonic_mode",
		issueNumber: 8489,
	},
	{
		oldName:     "tags",
		newName:     "host_metadata::tags",
		issueNumber: 9099,
	},
	{
		oldName:     "send_metadata",
		newName:     "host_metadata::enabled",
		issueNumber: 9099,
	},
	{
		oldName:     "use_resource_metadata",
		newName:     "host_metadata::hostname_source",
		issueNumber: 9099,
	},
	{
		oldName:     "metrics::report_quantiles",
		newName:     "metrics::summaries::mode",
		issueNumber: 8845,
	},
	{
		oldName:     "metrics::instrumentation_library_metadata_as_tags",
		newName:     "metrics::instrumentation_scope_as_tags",
		issueNumber: 11135,
	},
}

// Error implements the error interface.
func (e renameError) Error() string {
	return fmt.Sprintf(
		"%q was removed in favor of %q. See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/%d",
		e.oldName,
		e.newName,
		e.issueNumber,
	)
}

func handleRemovedSettings(configMap *confmap.Conf) (err error) {
	for _, removedErr := range removedSettings {
		if configMap.IsSet(removedErr.oldName) {
			err = multierr.Append(err, removedErr)
		}
	}
	return
}

var _ confmap.Unmarshaler = (*Config)(nil)

// Unmarshal a configuration map into the configuration struct.
func (c *Config) Unmarshal(configMap *confmap.Conf) error {
	if err := handleRemovedSettings(configMap); err != nil {
		return err
	}

	err := configMap.Unmarshal(c, confmap.WithErrorUnused())
	if err != nil {
		return err
	}

	c.API.Key = configopaque.String(strings.TrimSpace(string(c.API.Key)))

	// If an endpoint is not explicitly set, override it based on the site.
	if !configMap.IsSet("metrics::endpoint") {
		c.Metrics.TCPAddr.Endpoint = fmt.Sprintf("https://api.%s", c.API.Site)
	}
	if !configMap.IsSet("traces::endpoint") {
		c.Traces.TCPAddr.Endpoint = fmt.Sprintf("https://trace.agent.%s", c.API.Site)
	}
	if !configMap.IsSet("logs::endpoint") {
		c.Logs.TCPAddr.Endpoint = fmt.Sprintf("https://http-intake.logs.%s", c.API.Site)
	}

	// Return an error if an endpoint is explicitly set to ""
	if c.Metrics.TCPAddr.Endpoint == "" || c.Traces.TCPAddr.Endpoint == "" || c.Logs.TCPAddr.Endpoint == "" {
		return errEmptyEndpoint
	}

	return nil
}
