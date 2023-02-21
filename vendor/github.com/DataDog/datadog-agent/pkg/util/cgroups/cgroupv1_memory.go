// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"math"
	"os"
	"strconv"
)

// When no memory limit is set, the Kernel returns a maximum value, being computed as:
// See https://unix.stackexchange.com/questions/420906/what-is-the-value-for-the-cgroups-limit-in-bytes-if-the-memory-is-not-restricte
var memoryUnlimitedValue = (uint64(math.MaxInt64) / uint64(os.Getpagesize())) * uint64(os.Getpagesize())

func (c *cgroupV1) GetMemoryStats(stats *MemoryStats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	if !c.controllerMounted("memory") {
		return &ControllerNotFoundError{Controller: "memory"}
	}

	if err := parse2ColumnStats(c.fr, c.pathFor("memory", "memory.stat"), 0, 1, func(key, value string) error {
		intVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			reportError(newValueError(value, err))
			// Dont't stop parsing on a single faulty value
			return nil
		}

		switch key {
		case "total_cache":
			stats.Cache = &intVal
		case "total_swap":
			stats.Swap = &intVal
		case "total_rss":
			// Filter out aberrant values
			if intVal < 1<<63 {
				stats.RSS = &intVal
			}
		case "total_rss_huge":
			stats.RSSHuge = &intVal
		case "total_mapped_file":
			stats.MappedFile = &intVal
		case "total_pgpgin":
			stats.Pgpgin = &intVal
		case "total_pgpgout":
			stats.Pgpgout = &intVal
		case "total_pgfault":
			stats.Pgfault = &intVal
		case "total_pgmajfault":
			stats.Pgmajfault = &intVal
		case "total_inactive_anon":
			stats.InactiveAnon = &intVal
		case "total_active_anon":
			stats.ActiveAnon = &intVal
		case "total_inactive_file":
			stats.InactiveFile = &intVal
		case "total_active_file":
			stats.ActiveFile = &intVal
		case "total_unevictable":
			stats.Unevictable = &intVal
		case "hierarchical_memory_limit":
			stats.Limit = &intVal
		case "hierarchical_memsw_limit":
			stats.SwapLimit = &intVal
		}

		return nil
	}); err != nil {
		reportError(err)
	}

	if stats.Limit != nil && *stats.Limit >= memoryUnlimitedValue {
		stats.Limit = nil
	}
	if stats.SwapLimit != nil && *stats.SwapLimit >= memoryUnlimitedValue {
		stats.SwapLimit = nil
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory", "memory.usage_in_bytes"), &stats.UsageTotal); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory", "memory.failcnt"), &stats.OOMEvents); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory", "memory.kmem.usage_in_bytes"), &stats.KernelMemory); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory", "memory.soft_limit_in_bytes"), &stats.LowThreshold); err != nil {
		reportError(err)
	}
	if stats.LowThreshold != nil && *stats.LowThreshold >= memoryUnlimitedValue {
		stats.LowThreshold = nil
	}

	return nil
}
