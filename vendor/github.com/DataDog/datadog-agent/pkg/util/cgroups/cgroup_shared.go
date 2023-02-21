// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"strconv"
	"strings"
)

const (
	cgroupProcsFile = "cgroup.procs"
	procCgroupFile  = "cgroup"
)

// ParseCPUSetFormat counts CPUs in CPUSet specs like "0,1,5-8". These are comma-separated lists
// of processor IDs, with hyphenated ranges representing closed sets.
// So "0,1,5-8" represents processors 0, 1, 5, 6, 7, 8.
// The function returns the count of CPUs, in this case 6.
func ParseCPUSetFormat(line string) uint64 {
	var numCPUs uint64

	lineSlice := strings.Split(line, ",")
	for _, l := range lineSlice {
		lineParts := strings.Split(l, "-")
		if len(lineParts) == 2 {
			p0, _ := strconv.Atoi(lineParts[0])
			p1, _ := strconv.Atoi(lineParts[1])
			numCPUs += uint64(p1 - p0 + 1)
		} else if len(lineParts) == 1 {
			numCPUs++
		}
	}

	return numCPUs
}

func nilIfZero(value **uint64) {
	if *value != nil && **value == 0 {
		*value = nil
	}
}
