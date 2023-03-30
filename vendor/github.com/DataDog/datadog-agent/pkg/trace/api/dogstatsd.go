// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package api

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/trace/log"
)

// dogstatsdProxyHandler returns a new HTTP handler which will proxy requests to
// the DogStatsD endpoint in the Core Agent over UDP or UDS (defaulting to UDS
// if StatsdSocket is set in the *AgentConfig).
func (r *HTTPReceiver) dogstatsdProxyHandler() http.Handler {
	if !r.conf.StatsdEnabled {
		log.Info("DogstatsD disabled in the Agent configuration. The DogstatsD proxy endpoint will be non-functional.")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "503 Status Unavailable", http.StatusServiceUnavailable)
		})
	}
	if r.conf.StatsdPort == 0 {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Agent dogstatsd UDP port not configured, but required for dogstatsd proxy.", http.StatusServiceUnavailable)
		})
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(r.conf.StatsdHost, strconv.Itoa(r.conf.StatsdPort)))
	if err != nil {
		log.Errorf("Error resolving dogstatsd proxy addr to %s endpoint at %q: %v", "udp", addr, err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Failed to resolve dogstatsd address", http.StatusInternalServerError)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payloads := bytes.Split(body, []byte("\n"))

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Errorf("Error connecting to %s endpoint at %q: %v", "udp", addr, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		for _, p := range payloads {
			if _, err := conn.Write(p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	})
}
