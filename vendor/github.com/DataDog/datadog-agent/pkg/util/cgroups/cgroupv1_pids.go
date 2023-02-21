// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import "time"

func (c *cgroupV1) GetPIDStats(stats *PIDStats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	if !c.controllerMounted("pids") {
		return &ControllerNotFoundError{Controller: "pids"}
	}

	// In pids.current we get count of TIDs+PIDs
	if err := parseSingleUnsignedStat(c.fr, c.pathFor("pids", "pids.current"), &stats.HierarchicalThreadCount); err != nil {
		reportError(err)
	}

	if err := parseSingleUnsignedStat(c.fr, c.pathFor("pids", "pids.max"), &stats.HierarchicalThreadLimit); err != nil {
		reportError(err)
	}

	return nil
}

func (c *cgroupV1) GetPIDs(cacheValidity time.Duration) ([]int, error) {
	return c.pidMapper.getPIDsForCgroup(c.identifier, c.path, cacheValidity), nil
}
