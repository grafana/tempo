// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
)

// ErrMissingAPIKey is returned when the config could not be validated due to missing API key.
var ErrMissingAPIKey = errors.New("you must specify an API Key, either via a configuration file or the DD_API_KEY env var")

// Endpoint specifies an endpoint that the trace agent will write data (traces, stats & services) to.
type Endpoint struct {
	APIKey string `json:"-"` // never marshal this
	Host   string

	// NoProxy will be set to true when the proxy setting for the trace API endpoint
	// needs to be ignored (e.g. it is part of the "no_proxy" list in the yaml settings).
	NoProxy bool
}

// TelemetryEndpointPrefix specifies the prefix of the telemetry endpoint URL.
const TelemetryEndpointPrefix = "https://instrumentation-telemetry-intake."

// App Services env var
const azureAppServices = "DD_AZURE_APP_SERVICES"

// OTLP holds the configuration for the OpenTelemetry receiver.
type OTLP struct {
	// BindHost specifies the host to bind the receiver to.
	BindHost string `mapstructure:"-"`

	// GRPCPort specifies the port to use for the plain HTTP receiver.
	// If unset (or 0), the receiver will be off.
	GRPCPort int `mapstructure:"grpc_port"`

	// SpanNameRemappings is the map of datadog span names and preferred name to map to. This can be used to
	// automatically map Datadog Span Operation Names to an updated value. All entries should be key/value pairs.
	SpanNameRemappings map[string]string `mapstructure:"span_name_remappings"`

	// SpanNameAsResourceName specifies whether the OpenTelemetry span's name should be
	// used as the Datadog span's operation name. By default (when this is false), the
	// operation name is deduced from a combination between the instrumentation scope
	// name and the span kind.
	//
	// For context, the OpenTelemetry 'Span Name' is equivalent to the Datadog 'resource name'.
	// The Datadog Span's Operation Name equivalent in OpenTelemetry does not exist, but the span's
	// kind comes close.
	SpanNameAsResourceName bool `mapstructure:"span_name_as_resource_name"`

	// MaxRequestBytes specifies the maximum number of bytes that will be read
	// from an incoming HTTP request.
	MaxRequestBytes int64 `mapstructure:"-"`

	// UsePreviewHostnameLogic specifies wether to use the 'preview' OpenTelemetry attributes to hostname rules,
	// controlled in the Datadog exporter by the `exporter.datadog.hostname.preview` feature flag.
	// The 'preview' rules change the canonical hostname chosen in cloud providers to be consistent with the
	// one sent by Datadog cloud integrations.
	UsePreviewHostnameLogic bool `mapstructure:"-"`

	// ProbabilisticSampling specifies the percentage of traces to ingest. Exceptions are made for errors
	// and rare traces (outliers) if "RareSamplerEnabled" is true. Invalid values are equivalent to 100.
	// If spans have the "sampling.priority" attribute set, probabilistic sampling is skipped and the user's
	// decision is followed.
	ProbabilisticSampling float64
}

// ObfuscationConfig holds the configuration for obfuscating sensitive data
// for various span types.
type ObfuscationConfig struct {
	// ES holds the obfuscation configuration for ElasticSearch bodies.
	ES JSONObfuscationConfig `mapstructure:"elasticsearch"`

	// Mongo holds the obfuscation configuration for MongoDB queries.
	Mongo JSONObfuscationConfig `mapstructure:"mongodb"`

	// SQLExecPlan holds the obfuscation configuration for SQL Exec Plans. This is strictly for safety related obfuscation,
	// not normalization. Normalization of exec plans is configured in SQLExecPlanNormalize.
	SQLExecPlan JSONObfuscationConfig `mapstructure:"sql_exec_plan"`

	// SQLExecPlanNormalize holds the normalization configuration for SQL Exec Plans.
	SQLExecPlanNormalize JSONObfuscationConfig `mapstructure:"sql_exec_plan_normalize"`

	// HTTP holds the obfuscation settings for HTTP URLs.
	HTTP HTTPObfuscationConfig `mapstructure:"http"`

	// RemoveStackTraces specifies whether stack traces should be removed.
	// More specifically "error.stack" tag values will be cleared.
	RemoveStackTraces bool `mapstructure:"remove_stack_traces"`

	// Redis holds the configuration for obfuscating the "redis.raw_command" tag
	// for spans of type "redis".
	Redis Enablable `mapstructure:"redis"`

	// Memcached holds the configuration for obfuscating the "memcached.command" tag
	// for spans of type "memcached".
	Memcached Enablable `mapstructure:"memcached"`

	// CreditCards holds the configuration for obfuscating credit cards.
	CreditCards CreditCardsConfig `mapstructure:"credit_cards"`
}

// Export returns an obfuscate.Config matching o.
func (o *ObfuscationConfig) Export(conf *AgentConfig) obfuscate.Config {
	return obfuscate.Config{
		SQL: obfuscate.SQLConfig{
			TableNames:       conf.HasFeature("table_names"),
			ReplaceDigits:    conf.HasFeature("quantize_sql_tables") || conf.HasFeature("replace_sql_digits"),
			KeepSQLAlias:     conf.HasFeature("keep_sql_alias"),
			DollarQuotedFunc: conf.HasFeature("dollar_quoted_func"),
			Cache:            conf.HasFeature("sql_cache"),
		},
		ES: obfuscate.JSONConfig{
			Enabled:            o.ES.Enabled,
			KeepValues:         o.ES.KeepValues,
			ObfuscateSQLValues: o.ES.ObfuscateSQLValues,
		},
		Mongo: obfuscate.JSONConfig{
			Enabled:            o.Mongo.Enabled,
			KeepValues:         o.Mongo.KeepValues,
			ObfuscateSQLValues: o.Mongo.ObfuscateSQLValues,
		},
		SQLExecPlan: obfuscate.JSONConfig{
			Enabled:            o.SQLExecPlan.Enabled,
			KeepValues:         o.SQLExecPlan.KeepValues,
			ObfuscateSQLValues: o.SQLExecPlan.ObfuscateSQLValues,
		},
		SQLExecPlanNormalize: obfuscate.JSONConfig{
			Enabled:            o.SQLExecPlanNormalize.Enabled,
			KeepValues:         o.SQLExecPlanNormalize.KeepValues,
			ObfuscateSQLValues: o.SQLExecPlanNormalize.ObfuscateSQLValues,
		},
		HTTP: obfuscate.HTTPConfig{
			RemoveQueryString: o.HTTP.RemoveQueryString,
			RemovePathDigits:  o.HTTP.RemovePathDigits,
		},
		Logger: new(debugLogger),
	}
}

type debugLogger struct{}

func (debugLogger) Debugf(format string, params ...interface{}) {
	log.Debugf(format, params...)
}

// CreditCardsConfig holds the configuration for credit card obfuscation in
// (Meta) tags.
type CreditCardsConfig struct {
	// Enabled specifies whether this feature should be enabled.
	Enabled bool `mapstructure:"enabled"`

	// Luhn specifies whether Luhn checksum validation should be enabled.
	// https://dev.to/shiraazm/goluhn-a-simple-library-for-generating-calculating-and-verifying-luhn-numbers-588j
	// It reduces false positives, but increases the CPU time X3.
	Luhn bool `mapstructure:"luhn"`
}

// HTTPObfuscationConfig holds the configuration settings for HTTP obfuscation.
type HTTPObfuscationConfig struct {
	// RemoveQueryStrings determines query strings to be removed from HTTP URLs.
	RemoveQueryString bool `mapstructure:"remove_query_string" json:"remove_query_string"`

	// RemovePathDigits determines digits in path segments to be obfuscated.
	RemovePathDigits bool `mapstructure:"remove_paths_with_digits" json:"remove_path_digits"`
}

// Enablable can represent any option that has an "enabled" boolean sub-field.
type Enablable struct {
	Enabled bool `mapstructure:"enabled"`
}

// TelemetryConfig holds Instrumentation telemetry Endpoints information
type TelemetryConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	Endpoints []*Endpoint
}

// JSONObfuscationConfig holds the obfuscation configuration for sensitive
// data found in JSON objects.
type JSONObfuscationConfig struct {
	// Enabled will specify whether obfuscation should be enabled.
	Enabled bool `mapstructure:"enabled"`

	// KeepValues will specify a set of keys for which their values will
	// not be obfuscated.
	KeepValues []string `mapstructure:"keep_values"`

	// ObfuscateSQLValues will specify a set of keys for which their values
	// will be passed through SQL obfuscation
	ObfuscateSQLValues []string `mapstructure:"obfuscate_sql_values"`
}

// ReplaceRule specifies a replace rule.
type ReplaceRule struct {
	// Name specifies the name of the tag that the replace rule addresses. However,
	// some exceptions apply such as:
	// • "resource.name" will target the resource
	// • "*" will target all tags and the resource
	Name string `mapstructure:"name"`

	// Pattern specifies the regexp pattern to be used when replacing. It must compile.
	Pattern string `mapstructure:"pattern"`

	// Re holds the compiled Pattern and is only used internally.
	Re *regexp.Regexp `mapstructure:"-"`

	// Repl specifies the replacement string to be used when Pattern matches.
	Repl string `mapstructure:"repl"`
}

// WriterConfig specifies configuration for an API writer.
type WriterConfig struct {
	// ConnectionLimit specifies the maximum number of concurrent outgoing
	// connections allowed for the sender.
	ConnectionLimit int `mapstructure:"connection_limit"`

	// QueueSize specifies the maximum number or payloads allowed to be queued
	// in the sender.
	QueueSize int `mapstructure:"queue_size"`

	// FlushPeriodSeconds specifies the frequency at which the writer's buffer
	// will be flushed to the sender, in seconds. Fractions are permitted.
	FlushPeriodSeconds float64 `mapstructure:"flush_period_seconds"`
}

// FargateOrchestratorName is a Fargate orchestrator name.
type FargateOrchestratorName string

const (
	// OrchestratorECS represents AWS ECS
	OrchestratorECS FargateOrchestratorName = "ECS"
	// OrchestratorEKS represents AWS EKS
	OrchestratorEKS FargateOrchestratorName = "EKS"
	// OrchestratorUnknown is used when we cannot retrieve the orchestrator
	OrchestratorUnknown FargateOrchestratorName = "Unknown"
)

// ProfilingProxyConfig ...
type ProfilingProxyConfig struct {
	// DDURL ...
	DDURL string
	// AdditionalEndpoints ...
	AdditionalEndpoints map[string][]string
}

// EVPProxy contains the settings for the EVPProxy proxy.
type EVPProxy struct {
	// Enabled reports whether EVPProxy is enabled (true by default).
	Enabled bool
	// DDURL is the Datadog site to forward payloads to (defaults to the Site setting if not set).
	DDURL string
	// APIKey is the main API Key (defaults to the main API key).
	APIKey string `json:"-"` // Never marshal this field
	// ApplicationKey to be used for requests with the X-Datadog-NeedsAppKey set (defaults to the top-level Application Key).
	ApplicationKey string `json:"-"` // Never marshal this field
	// AdditionalEndpoints is a map of additional Datadog sites to API keys.
	AdditionalEndpoints map[string][]string
	// MaxPayloadSize indicates the size at which payloads will be rejected, in bytes.
	MaxPayloadSize int64
}

// DebuggerProxyConfig ...
type DebuggerProxyConfig struct {
	// DDURL ...
	DDURL string
	// APIKey ...
	APIKey string `json:"-"` // Never marshal this field
}

// AgentConfig handles the interpretation of the configuration (with default
// behaviors) in one place. It is also a simple structure to share across all
// the Agent components, with 100% safe and reliable values.
// It is exposed with expvar, so make sure to exclude any sensible field
// from JSON encoding. Use New() to create an instance.
type AgentConfig struct {
	Features map[string]struct{}

	Enabled      bool
	AgentVersion string
	GitCommit    string
	Site         string // the intake site to use (e.g. "datadoghq.com")

	// FargateOrchestrator specifies the name of the Fargate orchestrator. e.g. "ECS", "EKS", "Unknown"
	FargateOrchestrator FargateOrchestratorName

	// Global
	Hostname   string
	DefaultEnv string // the traces will default to this environment
	ConfigPath string // the source of this config, if any

	// Endpoints specifies the set of hosts and API keys where traces and stats
	// will be uploaded to. The first endpoint is the main configuration endpoint;
	// any following ones are read from the 'additional_endpoints' parts of the
	// configuration file, if present.
	Endpoints []*Endpoint

	// Concentrator
	BucketInterval   time.Duration // the size of our pre-aggregation per bucket
	ExtraAggregators []string

	// Sampler configuration
	ExtraSampleRate float64
	TargetTPS       float64
	ErrorTPS        float64
	MaxEPS          float64
	MaxRemoteTPS    float64

	// Rare Sampler configuration
	RareSamplerEnabled        bool
	RareSamplerTPS            int
	RareSamplerCooldownPeriod time.Duration
	RareSamplerCardinality    int

	// Receiver
	ReceiverHost    string
	ReceiverPort    int
	ReceiverSocket  string // if not empty, UDS will be enabled on unix://<receiver_socket>
	ConnectionLimit int    // for rate-limiting, how many unique connections to allow in a lease period (30s)
	ReceiverTimeout int
	MaxRequestBytes int64 // specifies the maximum allowed request size for incoming trace payloads

	WindowsPipeName        string
	PipeBufferSize         int
	PipeSecurityDescriptor string

	GUIPort string // the port of the Datadog Agent GUI (for control access)

	// Writers
	SynchronousFlushing     bool // Mode where traces are only submitted when FlushAsync is called, used for Serverless Extension
	StatsWriter             *WriterConfig
	TraceWriter             *WriterConfig
	ConnectionResetInterval time.Duration // frequency at which outgoing connections are reset. 0 means no reset is performed

	// internal telemetry
	StatsdEnabled  bool
	StatsdHost     string
	StatsdPort     int
	StatsdPipeName string // for Windows Pipes
	StatsdSocket   string // for UDS Sockets

	// logging
	LogFilePath   string
	LogThrottling bool

	// watchdog
	MaxMemory        float64       // MaxMemory is the threshold (bytes allocated) above which program panics and exits, to be restarted
	MaxCPU           float64       // MaxCPU is the max UserAvg CPU the program should consume
	WatchdogInterval time.Duration // WatchdogInterval is the delay between 2 watchdog checks

	// http/s proxying
	ProxyURL          *url.URL
	SkipSSLValidation bool

	// filtering
	Ignore map[string][]string

	// ReplaceTags is used to filter out sensitive information from tag values.
	// It maps tag keys to a set of replacements. Only supported in A6.
	ReplaceTags []*ReplaceRule

	// GlobalTags list metadata that will be added to all spans
	GlobalTags map[string]string

	// transaction analytics
	AnalyzedRateByServiceLegacy map[string]float64
	AnalyzedSpansByService      map[string]map[string]float64

	// infrastructure agent binary
	DDAgentBin string

	// Obfuscation holds sensitive data obufscator's configuration.
	Obfuscation *ObfuscationConfig

	// MaxResourceLen the maximum length the resource can have
	MaxResourceLen int

	// RequireTags specifies a list of tags which must be present on the root span in order for a trace to be accepted.
	RequireTags []*Tag

	// RejectTags specifies a list of tags which must be absent on the root span in order for a trace to be accepted.
	RejectTags []*Tag

	// OTLPReceiver holds the configuration for OpenTelemetry receiver.
	OTLPReceiver *OTLP

	// ProfilingProxy specifies settings for the profiling proxy.
	ProfilingProxy ProfilingProxyConfig

	// Telemetry settings
	TelemetryConfig *TelemetryConfig

	// EVPProxy contains the settings for the EVPProxy proxy.
	EVPProxy EVPProxy

	// DebuggerProxy contains the settings for the Live Debugger proxy.
	DebuggerProxy DebuggerProxyConfig

	// Proxy specifies a function to return a proxy for a given Request.
	// See (net/http.Transport).Proxy for more details.
	Proxy func(*http.Request) (*url.URL, error) `json:"-"`

	// MaxCatalogEntries specifies the maximum number of services to be added to the priority sampler's
	// catalog. If not set (0) it will default to 5000.
	MaxCatalogEntries int

	// RemoteSamplingClient retrieves sampling updates from the remote config backend
	RemoteSamplingClient RemoteClient `json:"-"`

	// ContainerTags ...
	ContainerTags func(cid string) ([]string, error) `json:"-"`

	// ContainerProcRoot is the root dir for `proc` info
	ContainerProcRoot string

	// Azure App Services
	InAzureAppServices bool

	// DebugServerPort defines the port used by the debug server
	DebugServerPort int
}

// RemoteClient client is used to APM Sampling Updates from a remote source.
// This is an interface around the client provided by pkg/config/remote to allow for easier testing.
type RemoteClient interface {
	Close()
	Start()
	RegisterAPMUpdate(func(update map[string]state.APMSamplingConfig))
}

// Tag represents a key/value pair.
type Tag struct {
	K, V string
}

// New returns a configuration with the default values.
func New() *AgentConfig {
	return &AgentConfig{
		Enabled:             true,
		DefaultEnv:          "none",
		Endpoints:           []*Endpoint{{Host: "https://trace.agent.datadoghq.com"}},
		FargateOrchestrator: OrchestratorUnknown,
		Site:                "datadoghq.com",
		MaxCatalogEntries:   5000,

		BucketInterval: time.Duration(10) * time.Second,

		ExtraSampleRate: 1.0,
		TargetTPS:       10,
		ErrorTPS:        10,
		MaxEPS:          200,
		MaxRemoteTPS:    100,

		RareSamplerEnabled:        false,
		RareSamplerTPS:            5,
		RareSamplerCooldownPeriod: 5 * time.Minute,
		RareSamplerCardinality:    200,

		ReceiverHost:           "localhost",
		ReceiverPort:           8126,
		MaxRequestBytes:        25 * 1024 * 1024, // 25MB
		PipeBufferSize:         1_000_000,
		PipeSecurityDescriptor: "D:AI(A;;GA;;;WD)",
		GUIPort:                "5002",

		StatsWriter:             new(WriterConfig),
		TraceWriter:             new(WriterConfig),
		ConnectionResetInterval: 0, // disabled

		StatsdHost:    "localhost",
		StatsdPort:    8125,
		StatsdEnabled: true,

		LogThrottling: true,

		MaxMemory:        5e8, // 500 Mb, should rarely go above 50 Mb
		MaxCPU:           0.5, // 50%, well behaving agents keep below 5%
		WatchdogInterval: 10 * time.Second,

		Ignore:                      make(map[string][]string),
		AnalyzedRateByServiceLegacy: make(map[string]float64),
		AnalyzedSpansByService:      make(map[string]map[string]float64),
		Obfuscation:                 &ObfuscationConfig{},
		MaxResourceLen:              5000,

		GlobalTags: make(map[string]string),

		Proxy:         http.ProxyFromEnvironment,
		OTLPReceiver:  &OTLP{},
		ContainerTags: noopContainerTagsFunc,
		TelemetryConfig: &TelemetryConfig{
			Endpoints: []*Endpoint{{Host: TelemetryEndpointPrefix + "datadoghq.com"}},
		},
		EVPProxy: EVPProxy{
			Enabled:        true,
			MaxPayloadSize: 5 * 1024 * 1024,
		},

		InAzureAppServices: inAzureAppServices(os.Getenv),

		Features: make(map[string]struct{}),
	}
}

func noopContainerTagsFunc(_ string) ([]string, error) {
	return nil, errors.New("ContainerTags function not defined")
}

// APIKey returns the first (main) endpoint's API key.
func (c *AgentConfig) APIKey() string {
	if len(c.Endpoints) == 0 {
		return ""
	}
	return c.Endpoints[0].APIKey
}

// NewHTTPClient returns a new http.Client to be used for outgoing connections to the
// Datadog API.
func (c *AgentConfig) NewHTTPClient() *ResetClient {
	return NewResetClient(c.ConnectionResetInterval, func() *http.Client {
		return &http.Client{
			Timeout:   10 * time.Second,
			Transport: c.NewHTTPTransport(),
		}
	})
}

// NewHTTPTransport returns a new http.Transport to be used for outgoing connections to
// the Datadog API.
func (c *AgentConfig) NewHTTPTransport() *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.SkipSSLValidation},
		// below field values are from http.DefaultTransport (go1.12)
		Proxy: c.Proxy,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return transport
}

func (c *AgentConfig) HasFeature(feat string) bool {
	_, ok := c.Features[feat]
	return ok
}

func (c *AgentConfig) AllFeatures() []string {
	feats := []string{}
	for feat := range c.Features {
		feats = append(feats, feat)
	}
	return feats
}

func inAzureAppServices(getenv func(string) string) bool {
	str := getenv(azureAppServices)
	if val, err := strconv.ParseBool(str); err == nil {
		return val
	} else {
		return false
	}
}
