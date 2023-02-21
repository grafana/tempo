// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows
// +build !windows

package api

import (
	"errors"
	"net"
)

func listenPipe(_, _ string, _ int) (net.Listener, error) {
	return nil, errors.New("Windows named pipes are only supported on Windows operating systems")
}
