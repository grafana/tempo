// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/trace/config"

	"go.uber.org/atomic"
)

// Error codes associated with each startup error
// The full list, and associated description is contained in the Tracking APM Onboarding RFC
const (
	GenericError               = 1
	CantCreateLogger           = 8
	TraceAgentNotEnabled       = 9
	CantWritePIDFile           = 10
	CantSetupAutoExit          = 11
	CantConfigureDogstatsd     = 12
	CantCreateRCCLient         = 13
	CantStartHttpServer        = 14
	CantStartUdsServer         = 15
	CantStartWindowsPipeServer = 16
	InvalidIntakeEndpoint      = 17
)

// OnboardingEvent contains
type OnboardingEvent struct {
	RequestType string                 `json:"request_type"`
	ApiVersion  string                 `json:"api_version"`
	Payload     OnboardingEventPayload `json:"payload,omitempty"`
}

// OnboardingEventPayload ...
type OnboardingEventPayload struct {
	EventName string               `json:"event_name"`
	Tags      OnboardingEventTags  `json:"tags"`
	Error     OnboardingEventError `json:"error,omitempty"`
}

// OnboardingEventTags ...
type OnboardingEventTags struct {
	AgentPlatform string `json:"agent_platform,omitempty"`
	AgentVersion  string `json:"agent_version,omitempty"`
	AgentHostname string `json:"agent_hostname,omitempty"`
	Env           string `json:"env,omitempty"`
}

// OnboardingEventError ...
type OnboardingEventError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// TelemetryCollector is the interface used to send reports about startup to the instrumentation telemetry intake
type TelemetryCollector interface {
	SendStartupSuccess()
	SendStartupError(code int, err error)
}

type telemetryCollector struct {
	client *config.ResetClient

	endpoints             []config.Endpoint
	userAgent             string
	cfg                   *config.AgentConfig
	collectedStartupError *atomic.Bool
}

// NewCollector returns either collector, or a noop implementation if instrumentation telemetry is disabled
func NewCollector(cfg *config.AgentConfig) TelemetryCollector {
	if !cfg.TelemetryConfig.Enabled {
		return &noopTelemetryCollector{}
	}

	var endpoints []config.Endpoint
	for _, endpoint := range cfg.TelemetryConfig.Endpoints {
		u, err := url.Parse(endpoint.Host)
		if err != nil {
			continue
		}
		u.Path = "/api/v2/apmtelemetry"
		endpointWithPath := *endpoint
		endpointWithPath.Host = u.String()

		endpoints = append(endpoints, endpointWithPath)
	}

	return &telemetryCollector{
		client:    cfg.NewHTTPClient(),
		endpoints: endpoints,
		userAgent: fmt.Sprintf("Datadog Trace Agent/%s/%s", cfg.AgentVersion, cfg.GitCommit),

		cfg:                   cfg,
		collectedStartupError: &atomic.Bool{},
	}
}

// NewNoopCollector returns a noop collector
func NewNoopCollector() TelemetryCollector {
	return &noopTelemetryCollector{}
}

func (f *telemetryCollector) sendEvent(event *OnboardingEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	bodyLen := strconv.Itoa(len(body))
	for _, endpoint := range f.endpoints {
		req, err := http.NewRequest("POST", endpoint.Host, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("User-Agent", f.userAgent)
		req.Header.Add("DD-Api-Key", endpoint.APIKey)
		req.Header.Add("Content-Length", bodyLen)

		resp, err := f.client.Do(req)
		if err != nil {
			continue
		}
		// Unconditionally read the body and ignore any errors
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func newOnboardingTelemetryPayload(config *config.AgentConfig) OnboardingEvent {
	return OnboardingEvent{
		RequestType: "apm-onboarding-event",
		ApiVersion:  "v1",
		Payload: OnboardingEventPayload{
			Tags: OnboardingEventTags{
				AgentVersion:  config.AgentVersion,
				AgentHostname: config.Hostname,
				Env:           config.DefaultEnv,
			},
		},
	}
}

func (f *telemetryCollector) SendStartupSuccess() {
	if f.collectedStartupError.Load() {
		return
	}
	ev := newOnboardingTelemetryPayload(f.cfg)
	ev.Payload.EventName = "agent.startup.success"
	f.sendEvent(&ev)
}

func (f *telemetryCollector) SendStartupError(code int, err error) {
	f.collectedStartupError.Store(true)
	ev := newOnboardingTelemetryPayload(f.cfg)
	ev.Payload.EventName = "agent.startup.error"
	ev.Payload.Error.Code = code
	ev.Payload.Error.Message = err.Error()
	f.sendEvent(&ev)
}

type noopTelemetryCollector struct{}

func (*noopTelemetryCollector) SendStartupSuccess()                  {}
func (*noopTelemetryCollector) SendStartupError(code int, err error) {}
