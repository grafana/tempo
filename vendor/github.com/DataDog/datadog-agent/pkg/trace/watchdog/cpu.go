// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows && !aix
// +build !windows,!aix

package watchdog

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/shirou/gopsutil/v3/process"
)

func getpid() int {
	// Based on gopsutil's HostProc https://github.com/shirou/gopsutil/blob/672e2518f2ce365ab8504c9f1a8038dc3ad09cf6/internal/common/common.go#L343-L345
	// This PID needs to match the one in the procfs that gopsutil is going to look in.
	p := os.Getenv("HOST_PROC")
	if p == "" {
		p = "/proc"
	}
	self := filepath.Join(p, "self")
	pidf, err := os.Readlink(self)
	if err != nil {
		log.Warnf("Failed to read pid from %s: %s. Falling back to os.Getpid", self, err)
		return os.Getpid()
	}
	pid, err := strconv.Atoi(filepath.Base(pidf))
	if err != nil {
		log.Warnf("Failed to parse pid from %s: %s. Falling back to os.Getpid", pidf, err)
		return os.Getpid()
	}
	return pid
}

func cpuTimeUser(pid int32) (float64, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return 0, err
	}
	times, err := p.Times()
	if err != nil {
		return 0, err
	}
	return times.User, nil
}
