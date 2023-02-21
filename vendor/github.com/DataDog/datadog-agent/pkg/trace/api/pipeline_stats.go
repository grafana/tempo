// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
)

const (
	pipelineStatsURLSuffix = "/api/v0.1/pipeline_stats"
)

// pipelineStatsEndpoint returns the pipeline intake url and the corresponding API key.
func pipelineStatsEndpoint(cfg *config.AgentConfig) (url *url.URL, apiKey string, err error) {
	if e := cfg.Endpoints; len(e) == 0 || e[0].Host == "" || e[0].APIKey == "" {
		return nil, "", errors.New("config was not properly validated")
	}
	urlStr := cfg.Endpoints[0].Host + pipelineStatsURLSuffix
	url, err = url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing pipeline stats intake URL %q: %v", urlStr, err)
	}
	return url, cfg.Endpoints[0].APIKey, nil
}

// pipelineStatsProxyHandler returns a new HTTP handler which will proxy requests to the pipeline stats intake.
func (r *HTTPReceiver) pipelineStatsProxyHandler() http.Handler {
	target, key, err := pipelineStatsEndpoint(r.conf)
	if err != nil {
		log.Errorf("Failed to start pipeline stats proxy handler: %v", err)
		return pipelineStatsErrorHandler(err)
	}
	tags := fmt.Sprintf("host:%s,default_env:%s,agent_version:%s", r.conf.Hostname, r.conf.DefaultEnv, r.conf.AgentVersion)
	if orch := r.conf.FargateOrchestrator; orch != config.OrchestratorUnknown {
		tag := fmt.Sprintf("orchestrator:fargate_%s", strings.ToLower(string(orch)))
		tags = tags + "," + tag
	}
	return newPipelineStatsProxy(r.conf, target, key, tags)
}

func pipelineStatsErrorHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		msg := fmt.Sprintf("Pipeline stats forwarder is OFF: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	})
}

// newPipelineStatsProxy creates an http.ReverseProxy which forwards requests to the pipeline stats intake.
// The tags will be added as a header to all proxied requests.
func newPipelineStatsProxy(conf *config.AgentConfig, target *url.URL, key string, tags string) *httputil.ReverseProxy {
	cidProvider := NewIDProvider(conf.ContainerProcRoot)
	director := func(req *http.Request) {
		req.Header.Set("Via", fmt.Sprintf("trace-agent %s", conf.AgentVersion))
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to the default value
			// that net/http gives it: Go-http-client/1.1
			// See https://codereview.appspot.com/7532043
			req.Header.Set("User-Agent", "")
		}
		containerID := cidProvider.GetContainerID(req.Context(), req.Header)
		if ctags := getContainerTags(conf.ContainerTags, containerID); ctags != "" {
			req.Header.Set("X-Datadog-Container-Tags", ctags)
		}
		req.Header.Set("X-Datadog-Additional-Tags", tags)
		metrics.Count("datadog.trace_agent.pipelines_stats", 1, nil, 1)
		req.Host = target.Host
		req.URL = target
		req.Header.Set("DD-API-KEY", key)
	}
	logger := log.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director:  director,
		ErrorLog:  stdlog.New(logger, "pipeline_stats.Proxy: ", 0),
		Transport: conf.NewHTTPTransport(),
	}
}
