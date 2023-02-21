// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
)

// Endpoint specifies an API endpoint definition.
type Endpoint struct {
	// Pattern specifies the API pattern, as registered by the HTTP handler.
	Pattern string

	// Handler specifies the http.Handler for this endpoint.
	Handler func(*HTTPReceiver) http.Handler

	// Hidden reports whether this endpoint should be hidden in the /info
	// discovery endpoint.
	Hidden bool

	// IsEnabled specifies a function which reports whether this endpoint should be enabled
	// based on the given config conf.
	IsEnabled func(conf *config.AgentConfig) bool
}

// AttachEndpoint attaches an additional endpoint to the trace-agent. It is not thread-safe
// and should be called before (pkg/trace.*Agent).Run or (*HTTPReceiver).Start. In other
// words, endpoint setup must be final before the agent or HTTP receiver starts.
func AttachEndpoint(e Endpoint) { endpoints = append(endpoints, e) }

// endpoints specifies the list of endpoints registered for the trace-agent API.
var endpoints = []Endpoint{
	{
		Pattern: "/spans",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v01, r.handleTraces) },
		Hidden:  true,
	},
	{
		Pattern: "/services",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v01, r.handleServices) },
		Hidden:  true,
	},
	{
		Pattern: "/v0.1/spans",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v01, r.handleTraces) },
		Hidden:  true,
	},
	{
		Pattern: "/v0.1/services",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v01, r.handleServices) },
		Hidden:  true,
	},
	{
		Pattern: "/v0.2/traces",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v02, r.handleTraces) },
		Hidden:  true,
	},
	{
		Pattern: "/v0.2/services",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v02, r.handleServices) },
		Hidden:  true,
	},
	{
		Pattern: "/v0.3/traces",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v03, r.handleTraces) },
	},
	{
		Pattern: "/v0.3/services",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v03, r.handleServices) },
	},
	{
		Pattern: "/v0.4/traces",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v04, r.handleTraces) },
	},
	{
		Pattern: "/v0.4/services",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v04, r.handleServices) },
	},
	{
		Pattern: "/v0.5/traces",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(v05, r.handleTraces) },
	},
	{
		Pattern: "/v0.7/traces",
		Handler: func(r *HTTPReceiver) http.Handler { return r.handleWithVersion(V07, r.handleTraces) },
	},
	{
		Pattern: "/profiling/v1/input",
		Handler: func(r *HTTPReceiver) http.Handler { return r.profileProxyHandler() },
	},
	{
		Pattern: "/telemetry/proxy/",
		Handler: func(r *HTTPReceiver) http.Handler {
			return http.StripPrefix("/telemetry/proxy", r.telemetryProxyHandler())
		},
		IsEnabled: func(cfg *config.AgentConfig) bool { return cfg.TelemetryConfig.Enabled },
	},
	{
		Pattern: "/v0.6/stats",
		Handler: func(r *HTTPReceiver) http.Handler { return http.HandlerFunc(r.handleStats) },
	},
	{
		Pattern: "/v0.1/pipeline_stats",
		Handler: func(r *HTTPReceiver) http.Handler { return r.pipelineStatsProxyHandler() },
	},
	{
		Pattern: "/evp_proxy/v1/",
		Handler: func(r *HTTPReceiver) http.Handler { return r.evpProxyHandler(1) },
	},
	{
		Pattern: "/evp_proxy/v2/",
		Handler: func(r *HTTPReceiver) http.Handler { return r.evpProxyHandler(2) },
	},
	{
		Pattern: "/evp_proxy/v3/",
		Handler: func(r *HTTPReceiver) http.Handler { return r.evpProxyHandler(2) },
	},
	{
		Pattern: "/debugger/v1/input",
		Handler: func(r *HTTPReceiver) http.Handler { return r.debuggerProxyHandler() },
	},
	{
		Pattern: "/dogstatsd/v1/proxy", // deprecated
		Handler: func(r *HTTPReceiver) http.Handler { return r.dogstatsdProxyHandler() },
	},
	{
		Pattern: "/dogstatsd/v2/proxy",
		Handler: func(r *HTTPReceiver) http.Handler { return r.dogstatsdProxyHandler() },
	},
}
