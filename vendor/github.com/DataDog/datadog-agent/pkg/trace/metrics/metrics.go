// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !benchmarking
// +build !benchmarking

// Package metrics exposes utilities for setting up and using a sub-set of Datadog's dogstatsd
// client.
package metrics

import (
	"errors"
	"net"
	"strconv"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
)

// findAddr finds the correct address to connect to the Dogstatsd server.
func findAddr(conf *config.AgentConfig) (string, error) {
	if conf.StatsdPort > 0 {
		// UDP enabled
		return net.JoinHostPort(conf.StatsdHost, strconv.Itoa(conf.StatsdPort)), nil
	}
	if conf.StatsdPipeName != "" {
		// Windows Pipes can be used
		return `\\.\pipe\` + conf.StatsdPipeName, nil
	}
	if conf.StatsdSocket != "" {
		// Unix sockets can be used
		return `unix://` + conf.StatsdSocket, nil
	}
	return "", errors.New("dogstatsd_port is set to 0 and no alternative is available")
}

// Configure creates a statsd client for the given agent's configuration, using the specified global tags.
func Configure(conf *config.AgentConfig, tags []string) error {
	addr, err := findAddr(conf)
	if err != nil {
		return err
	}
	client, err := statsd.New(addr, statsd.WithTags(tags))
	if err != nil {
		return err
	}
	Client = client
	return nil
}
