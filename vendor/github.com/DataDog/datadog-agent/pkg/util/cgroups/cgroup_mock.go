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
	ID          string
	Parent      Cgroup
	ParentError error
	CPU         *CPUStats
	CPUError    error
	Memory      *MemoryStats
	MemoryError error
	IOStats     *IOStats
	IOError     error
	PIDStats    *PIDStats
	PIDError    error
	PIDs        []int
	PIDsError   error
}

// Identifier mock
func (mc *MockCgroup) Identifier() string {
	return mc.ID
}

// GetParent mock
func (mc *MockCgroup) GetParent() (Cgroup, error) {
	return mc.Parent, mc.ParentError
}

// GetCPUStats mock
func (mc *MockCgroup) GetCPUStats(cpuStats *CPUStats) error {
	if mc.CPU != nil {
		*cpuStats = *mc.CPU
	}
	return mc.CPUError
}

// GetMemoryStats mock
func (mc *MockCgroup) GetMemoryStats(memoryStats *MemoryStats) error {
	if mc.Memory != nil {
		*memoryStats = *mc.Memory
	}
	return mc.MemoryError
}

// GetIOStats mock
func (mc *MockCgroup) GetIOStats(ioStats *IOStats) error {
	if mc.IOStats != nil {
		*ioStats = *mc.IOStats
	}
	return mc.IOError
}

// GetPIDStats mock
func (mc *MockCgroup) GetPIDStats(pidStats *PIDStats) error {
	if mc.PIDStats != nil {
		*pidStats = *mc.PIDStats
	}
	return mc.PIDError
}

// GetPIDs mock
func (mc *MockCgroup) GetPIDs(cacheValidity time.Duration) ([]int, error) {
	return mc.PIDs, mc.PIDsError
}
