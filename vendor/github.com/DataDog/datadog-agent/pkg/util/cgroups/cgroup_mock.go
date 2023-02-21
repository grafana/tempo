// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import "time"

// MockCgroup is a mock implementing the Cgroup interface
type MockCgroup struct {
	ID       string
	Parent   Cgroup
	CPU      *CPUStats
	Memory   *MemoryStats
	IOStats  *IOStats
	PIDStats *PIDStats
	PIDs     []int
	Error    error
}

// Identifier mock
func (mc *MockCgroup) Identifier() string {
	return mc.ID
}

// GetParent mock
func (mc *MockCgroup) GetParent() (Cgroup, error) {
	return mc.Parent, mc.Error
}

// GetStats mock
func (mc *MockCgroup) GetStats(stats *Stats) error {
	stats.CPU = mc.CPU
	stats.Memory = mc.Memory
	stats.IO = mc.IOStats
	stats.PID = mc.PIDStats
	return mc.Error
}

// GetCPUStats mock
func (mc *MockCgroup) GetCPUStats(cpuStats *CPUStats) error {
	if mc.CPU != nil {
		*cpuStats = *mc.CPU
	}
	return mc.Error
}

// GetMemoryStats mock
func (mc *MockCgroup) GetMemoryStats(memoryStats *MemoryStats) error {
	if mc.Memory != nil {
		*memoryStats = *mc.Memory
	}
	return mc.Error
}

// GetIOStats mock
func (mc *MockCgroup) GetIOStats(ioStats *IOStats) error {
	if mc.IOStats != nil {
		*ioStats = *mc.IOStats
	}
	return mc.Error
}

// GetPIDStats mock
func (mc *MockCgroup) GetPIDStats(pidStats *PIDStats) error {
	if mc.PIDStats != nil {
		*pidStats = *mc.PIDStats
	}
	return mc.Error
}

// GetPIDs mock
func (mc *MockCgroup) GetPIDs(cacheValidity time.Duration) ([]int, error) {
	return mc.PIDs, mc.Error
}
