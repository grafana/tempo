// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
func pipelineStatsEndpoints(cfg *config.AgentConfig) (urls []*url.URL, apiKeys []string, err error) {
	if e := cfg.Endpoints; len(e) == 0 || e[0].Host == "" || e[0].APIKey == "" {
		return nil, nil, errors.New("config was not properly validated")
	}
	for _, e := range cfg.Endpoints {
		urlStr := e.Host + pipelineStatsURLSuffix
		url, err := url.Parse(urlStr)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing pipeline stats intake URL %q: %v", urlStr, err)
		}
		urls = append(urls, url)
		apiKeys = append(apiKeys, e.APIKey)
	}
	return urls, apiKeys, nil
}

// pipelineStatsProxyHandler returns a new HTTP handler which will proxy requests to the pipeline stats intake.
func (r *HTTPReceiver) pipelineStatsProxyHandler() http.Handler {
	urls, apiKeys, err := pipelineStatsEndpoints(r.conf)
	if err != nil {
		log.Errorf("Failed to start pipeline stats proxy handler: %v", err)
		return pipelineStatsErrorHandler(err)
	}
	tags := fmt.Sprintf("host:%s,default_env:%s,agent_version:%s", r.conf.Hostname, r.conf.DefaultEnv, r.conf.AgentVersion)
	if orch := r.conf.FargateOrchestrator; orch != config.OrchestratorUnknown {
		tag := fmt.Sprintf("orchestrator:fargate_%s", strings.ToLower(string(orch)))
		tags = tags + "," + tag
	}
	return newPipelineStatsProxy(r.conf, urls, apiKeys, tags)
}

func pipelineStatsErrorHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		msg := fmt.Sprintf("Pipeline stats forwarder is OFF: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	})
}

// newPipelineStatsProxy creates an http.ReverseProxy which forwards requests to the pipeline stats intake.
// The tags will be added as a header to all proxied requests.
func newPipelineStatsProxy(conf *config.AgentConfig, urls []*url.URL, apiKeys []string, tags string) *httputil.ReverseProxy {
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
	}
	logger := log.NewThrottled(5, 10*time.Second) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director:  director,
		ErrorLog:  stdlog.New(logger, "pipeline_stats.Proxy: ", 0),
		Transport: &multiDataStreamsTransport{rt: conf.NewHTTPTransport(), targets: urls, keys: apiKeys},
	}
}

// multiDataStreamsTransport sends HTTP requests to multiple targets using an
// underlying http.RoundTripper. API keys are set separately for each target.
// When multiple endpoints are in use the response from the main endpoint
// is proxied back to the client, while for all additional endpoints the
// response is discarded. There is no de-duplication done between endpoint
// hosts or api keys.
type multiDataStreamsTransport struct {
	rt      http.RoundTripper
	targets []*url.URL
	keys    []string
}

func (m *multiDataStreamsTransport) RoundTrip(req *http.Request) (rresp *http.Response, rerr error) {
	setTarget := func(r *http.Request, u *url.URL, apiKey string) {
		r.Host = u.Host
		r.URL = u
		r.Header.Set("DD-API-KEY", apiKey)
	}
	if len(m.targets) == 1 {
		setTarget(req, m.targets[0], m.keys[0])
		return m.rt.RoundTrip(req)
	}
	slurp, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	for i, u := range m.targets {
		newreq := req.Clone(req.Context())
		newreq.Body = io.NopCloser(bytes.NewReader(slurp))
		setTarget(newreq, u, m.keys[i])
		if i == 0 {
			// given the way we construct the list of targets the main endpoint
			// will be the first one called, we return its response and error
			rresp, rerr = m.rt.RoundTrip(newreq)
			continue
		}

		if resp, err := m.rt.RoundTrip(newreq); err == nil {
			// we discard responses for all subsequent requests
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
		} else {
			log.Error(err)
		}
	}
	return rresp, rerr
}
