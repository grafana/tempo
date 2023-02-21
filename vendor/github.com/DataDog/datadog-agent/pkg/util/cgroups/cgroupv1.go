// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import "path/filepath"

type cgroupV1 struct {
	identifier  string
	mountPoints map[string]string
	path        string
	fr          fileReader
	pidMapper   pidMapper
}

func newCgroupV1(identifier, path string, mountPoints map[string]string, pidMapper pidMapper) *cgroupV1 {
	return &cgroupV1{
		identifier:  identifier,
		mountPoints: mountPoints,
		path:        path,
		pidMapper:   pidMapper,
		fr:          defaultFileReader,
	}
}

func (c *cgroupV1) Identifier() string {
	return c.identifier
}

func (c *cgroupV1) GetParent() (Cgroup, error) {
	parentPath := filepath.Join(c.path, "/..")
	return newCgroupV1(filepath.Base(parentPath), parentPath, c.mountPoints, c.pidMapper), nil
}

func (c *cgroupV1) GetStats(stats *Stats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	cpuStats := CPUStats{}
	err := c.GetCPUStats(&cpuStats)
	if err != nil {
		return err
	}
	stats.CPU = &cpuStats

	memoryStats := MemoryStats{}
	err = c.GetMemoryStats(&memoryStats)
	if err != nil {
		return err
	}
	stats.Memory = &memoryStats

	ioStats := IOStats{}
	err = c.GetIOStats(&ioStats)
	if err != nil {
		return err
	}
	stats.IO = &ioStats

	pidStats := PIDStats{}
	err = c.GetPIDStats(&pidStats)
	if err != nil {
		return err
	}
	stats.PID = &pidStats

	return nil
}

func (c *cgroupV1) controllerMounted(controller string) bool {
	_, found := c.mountPoints[controller]
	return found
}

// Expects controller to exist, see controllerMounted
func (c *cgroupV1) pathFor(controller, file string) string {
	return filepath.Join(c.mountPoints[controller], c.path, file)
}
