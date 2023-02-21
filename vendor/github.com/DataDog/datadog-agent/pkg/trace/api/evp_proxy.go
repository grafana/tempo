// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package api

import (
	"bytes"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/api/apiutil"
	"github.com/DataDog/datadog-agent/pkg/trace/api/internal/header"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/DataDog/datadog-agent/pkg/trace/metrics"
)

const (
	validSubdomainSymbols       = "_-."
	validPathSymbols            = "/_-+"
	validPathQueryStringSymbols = "/_-+@?&=.:\""
)

// allowedHeaders contains the headers that the proxy will forward. All others will be cleared.
var allowedHeaders = [...]string{"Content-Type", "User-Agent", "DD-CI-PROVIDER-NAME"}

// evpProxyEndpointsFromConfig returns the configured list of endpoints to forward payloads to.
func evpProxyEndpointsFromConfig(conf *config.AgentConfig) []config.Endpoint {
	apiKey := conf.EVPProxy.APIKey
	if apiKey == "" {
		apiKey = conf.APIKey()
	}
	endpoint := conf.EVPProxy.DDURL
	if endpoint == "" {
		endpoint = conf.Site
	}
	endpoints := []config.Endpoint{{Host: endpoint, APIKey: apiKey}} // main endpoint
	for host, keys := range conf.EVPProxy.AdditionalEndpoints {
		for _, key := range keys {
			endpoints = append(endpoints, config.Endpoint{
				Host:   host,
				APIKey: key,
			})
		}
	}
	return endpoints
}

func (r *HTTPReceiver) evpProxyHandler(apiVersion int) http.Handler {
	if !r.conf.EVPProxy.Enabled {
		return evpProxyErrorHandler("Has been disabled in config")
	}
	handler := evpProxyForwarder(r.conf)
	return http.StripPrefix(fmt.Sprintf("/evp_proxy/v%d", apiVersion), handler)
}

// evpProxyErrorHandler returns an HTTP handler that will always return
// http.StatusMethodNotAllowed along with a clarification.
func evpProxyErrorHandler(message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		msg := fmt.Sprintf("EVPProxy is disabled: %v", message)
		http.Error(w, msg, http.StatusMethodNotAllowed)
	})
}

// evpProxyForwarder creates an http.ReverseProxy which can forward payloads to
// one or more endpoints, based on the request received and the Agent configuration.
// Headers are not proxied, instead we add our own known set of headers.
// See also evpProxyTransport below.
func evpProxyForwarder(conf *config.AgentConfig) http.Handler {
	endpoints := evpProxyEndpointsFromConfig(conf)
	logger := stdlog.New(log.NewThrottled(5, 10*time.Second), "EVPProxy: ", 0) // limit to 5 messages every 10 seconds
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// The X-Forwarded-For header can be abused to fake the origin of requests and we don't need it,
			// so we set it to null to tell ReverseProxy to not set it.
			req.Header["X-Forwarded-For"] = nil
		},
		ErrorLog:  logger,
		Transport: &evpProxyTransport{conf.NewHTTPTransport(), endpoints, conf, NewIDProvider(conf.ContainerProcRoot)},
	}
}

// evpProxyTransport sends HTTPS requests to multiple targets using an
// underlying http.RoundTripper. API keys are set separately for each target.
// When multiple endpoints are in use the response from the first endpoint
// is proxied back to the client, while for all aditional endpoints the
// response is discarded.
type evpProxyTransport struct {
	transport           http.RoundTripper
	endpoints           []config.Endpoint
	conf                *config.AgentConfig
	containerIDProvider IDProvider
}

func (t *evpProxyTransport) RoundTrip(req *http.Request) (rresp *http.Response, rerr error) {
	if req.Body != nil && t.conf.EVPProxy.MaxPayloadSize > 0 {
		req.Body = apiutil.NewLimitedReader(req.Body, t.conf.EVPProxy.MaxPayloadSize)
	}
	start := time.Now()
	tags := []string{} // these tags are only for the debug metrics, not the payloads we forward
	if ct := req.Header.Get("Content-Type"); ct != "" {
		tags = append(tags, "content_type:"+ct)
	}
	defer func() {
		metrics.Count("datadog.trace_agent.evp_proxy.request", 1, tags, 1)
		metrics.Count("datadog.trace_agent.evp_proxy.request_bytes", req.ContentLength, tags, 1)
		metrics.Timing("datadog.trace_agent.evp_proxy.request_duration_ms", time.Since(start), tags, 1)
		if rerr != nil {
			metrics.Count("datadog.trace_agent.evp_proxy.request_error", 1, tags, 1)
		}
	}()

	subdomain := req.Header.Get("X-Datadog-EVP-Subdomain")
	containerID := t.containerIDProvider.GetContainerID(req.Context(), req.Header)
	needsAppKey := (strings.ToLower(req.Header.Get("X-Datadog-NeedsAppKey")) == "true")

	// Sanitize the input, don't accept any valid URL but just some limited subset
	if len(subdomain) == 0 {
		return nil, fmt.Errorf("EVPProxy: no subdomain specified")
	}
	if !isValidSubdomain(subdomain) {
		return nil, fmt.Errorf("EVPProxy: invalid subdomain: %s", subdomain)
	}
	tags = append(tags, "subdomain:"+subdomain)
	if !isValidPath(req.URL.Path) {
		return nil, fmt.Errorf("EVPProxy: invalid target path: %s", req.URL.Path)
	}
	if !isValidQueryString(req.URL.RawQuery) {
		return nil, fmt.Errorf("EVPProxy: invalid query string: %s", req.URL.RawQuery)
	}

	if needsAppKey && t.conf.EVPProxy.ApplicationKey == "" {
		return nil, fmt.Errorf("EVPProxy: ApplicationKey needed but not set")
	}

	// We don't want to forward arbitrary headers, create a copy of the input headers and clear them
	inputHeaders := req.Header
	req.Header = http.Header{}

	// Set standard headers
	req.Header.Set("User-Agent", "") // Set to empty string so Go doesn't set its default
	req.Header.Set("Via", fmt.Sprintf("trace-agent %s", t.conf.AgentVersion))

	// Copy allowed headers from the input request
	for _, header := range allowedHeaders {
		val := inputHeaders.Get(header)
		if val != "" {
			req.Header.Set(header, val)
		}
	}

	// Set Datadog headers, except API key which is set per-endpoint
	if containerID != "" {
		req.Header.Set(header.ContainerID, containerID)
		if ctags := getContainerTags(t.conf.ContainerTags, containerID); ctags != "" {
			req.Header.Set("X-Datadog-Container-Tags", ctags)
		}
	}
	req.Header.Set("X-Datadog-Hostname", t.conf.Hostname)
	req.Header.Set("X-Datadog-AgentDefaultEnv", t.conf.DefaultEnv)
	req.Header.Set(header.ContainerID, containerID)
	if needsAppKey {
		req.Header.Set("DD-APPLICATION-KEY", t.conf.EVPProxy.ApplicationKey)
	}

	// Set target URL and API key header (per domain)
	req.URL.Scheme = "https"
	setTarget := func(r *http.Request, host, apiKey string) {
		targetHost := subdomain + "." + host
		r.Host = targetHost
		r.URL.Host = targetHost
		r.Header.Set("DD-API-KEY", apiKey)
	}

	// Shortcut if we only have one endpoint
	if len(t.endpoints) == 1 {
		setTarget(req, t.endpoints[0].Host, t.endpoints[0].APIKey)
		return t.transport.RoundTrip(req)
	}

	// There's more than one destination endpoint
	var slurp []byte
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		slurp = body
	}
	for i, endpointDomain := range t.endpoints {
		newreq := req.Clone(req.Context())
		if slurp != nil {
			newreq.Body = io.NopCloser(bytes.NewReader(slurp))
		}
		setTarget(newreq, endpointDomain.Host, endpointDomain.APIKey)
		if i == 0 {
			// given the way we construct the list of targets the main endpoint
			// will be the first one called, we return its response and error
			rresp, rerr = t.transport.RoundTrip(newreq)
			continue
		}

		if resp, err := t.transport.RoundTrip(newreq); err == nil {
			// we discard responses for all subsequent requests
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
		} else {
			log.Error(err)
		}
	}
	return rresp, rerr
}

// We don't want to accept any valid URL, we are strict in what we accept as a subdomain, path
// or query string to prevent abusing this API for other purposes than forwarding evp payloads.
func isValidSubdomain(s string) bool {
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune(validSubdomainSymbols, c) {
			return false
		}
	}
	return true
}

func isValidPath(s string) bool {
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune(validPathSymbols, c) {
			return false
		}
	}
	return true
}

func isValidQueryString(s string) bool {
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune(validPathQueryStringSymbols, c) {
			return false
		}
	}
	return true
}
