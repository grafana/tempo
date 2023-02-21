// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"path/filepath"
)

type cgroupV2 struct {
	identifier   string
	cgroupRoot   string
	relativePath string
	controllers  map[string]struct{}
	fr           fileReader
	pidMapper    pidMapper
}

func newCgroupV2(identifier, cgroupRoot, relativePath string, controllers map[string]struct{}, pidMapper pidMapper) *cgroupV2 {
	return &cgroupV2{
		identifier:   identifier,
		cgroupRoot:   cgroupRoot,
		relativePath: relativePath,
		controllers:  controllers,
		pidMapper:    pidMapper,
		fr:           defaultFileReader,
	}
}

func (c *cgroupV2) Identifier() string {
	return c.identifier
}

func (c *cgroupV2) GetParent() (Cgroup, error) {
	parentPath := filepath.Join(c.relativePath, "/..")
	return newCgroupV2(filepath.Base(parentPath), c.cgroupRoot, parentPath, c.controllers, c.pidMapper), nil
}

func (c *cgroupV2) GetStats(stats *Stats) error {
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

func (c *cgroupV2) controllerActivated(controller string) bool {
	_, found := c.controllers[controller]
	return found
}

func (c *cgroupV2) pathFor(filename string) string {
	return filepath.Join(c.cgroupRoot, c.relativePath, filename)
}
