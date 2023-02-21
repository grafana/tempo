// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/util/pointer"
)

func (c *cgroupV2) GetMemoryStats(stats *MemoryStats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	if !c.controllerActivated("memory") {
		return &ControllerNotFoundError{Controller: "memory"}
	}

	var kernelStack, slab *uint64

	if err := parse2ColumnStats(c.fr, c.pathFor("memory.stat"), 0, 1, func(key, value string) error {
		intVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			reportError(newValueError(value, err))
			// Dont't stop parsing on a single faulty value
			return nil
		}

		switch key {
		case "file":
			stats.Cache = &intVal
		case "anon":
			stats.RSS = &intVal
		case "anon_thp":
			stats.RSSHuge = &intVal
		case "file_mapped":
			stats.MappedFile = &intVal
		case "pgfault":
			stats.Pgfault = &intVal
		case "pgmajfault":
			stats.Pgmajfault = &intVal
		case "inactive_anon":
			stats.InactiveAnon = &intVal
		case "active_anon":
			stats.ActiveAnon = &intVal
		case "inactive_file":
			stats.InactiveFile = &intVal
		case "active_file":
			stats.ActiveFile = &intVal
		case "unevictable":
			stats.Unevictable = &intVal
		case "kernel_stack":
			kernelStack = &intVal
		case "slab":
			slab = &intVal
		}

		return nil
	}); err != nil {
		reportError(err)
	}

	if kernelStack != nil && slab != nil {
		stats.KernelMemory = pointer.Ptr(*kernelStack + *slab)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.current"), &stats.UsageTotal); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.min"), &stats.MinThreshold); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.MinThreshold)

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.low"), &stats.LowThreshold); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.LowThreshold)

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.high"), &stats.HighThreshold); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.HighThreshold)

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.max"), &stats.Limit); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.Limit)

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.swap.current"), &stats.Swap); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.swap.high"), &stats.SwapHighThreshold); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.SwapHighThreshold)

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("memory.swap.max"), &stats.SwapLimit); err != nil {
		reportError(err)
	}
	nilIfZero(&stats.SwapLimit)

	if err := parse2ColumnStats(c.fr, c.pathFor("memory.events"), 0, 1, func(key, value string) error {
		intVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			reportError(newValueError(value, err))
			// Dont't stop parsing on a single faulty value
			return nil
		}

		switch key {
		case "oom":
			stats.OOMEvents = &intVal
		case "oom_kill":
			stats.OOMKiilEvents = &intVal
		}

		return nil
	}); err != nil {
		reportError(err)
	}

	if err := parsePSI(c.fr, c.pathFor("memory.pressure"), &stats.PSISome, &stats.PSIFull); err != nil {
		reportError(err)
	}

	return nil
}
